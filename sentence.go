package nmea

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

const (
	// SentenceStart is the token to indicate the start of a sentence.
	SentenceStart = "$"

	// SentenceStartEncapsulated is the token to indicate the start of encapsulated data.
	SentenceStartEncapsulated = "!"

	// FieldSep is the token to delimit fields of a sentence.
	FieldSep = ","

	// ChecksumSep is the token to delimit the checksum of a sentence.
	ChecksumSep = "*"
)

var (
	customParsersMu = &sync.Mutex{}
	customParsers   = map[string]ParserFunc{}
)

// ParserFunc callback used to parse specific sentence variants
type ParserFunc func(BaseSentence) (Sentence, error)

// Sentence interface for all NMEA sentence
type Sentence interface {
	fmt.Stringer
	Prefix() string
	DataType() string
	TalkerID() string
}

// BaseSentence contains the information about the NMEA sentence
type BaseSentence struct {
	Talker   string   // The talker id (e.g GP)
	Type     string   // The data type (e.g GSA)
	Fields   []string // Array of fields
	Raw      string   // The raw NMEA sentence received
	TagBlock TagBlock // NMEA tagblock
}

// Prefix returns the talker and type of message
func (s BaseSentence) Prefix() string {
	return s.Talker + s.Type
}

// DataType returns the type of the message
func (s BaseSentence) DataType() string {
	return s.Type
}

// TalkerID returns the talker of the message
func (s BaseSentence) TalkerID() string {
	return s.Talker
}

// String formats the sentence into a string
func (s BaseSentence) String() string { return s.Raw }

// parseSentence parses a raw message into it's fields
func parseSentence(raw string) (BaseSentence, error) {
	raw = strings.TrimSpace(raw)

	var (
		tagBlock TagBlock
	)

	startIndex := strings.IndexAny(raw, SentenceStart+SentenceStartEncapsulated)
	if startIndex != 0 {
		return BaseSentence{}, errors.New("nmea: sentence does not start with a '$' or '!'")
	}
	sumSepIndex := strings.Index(raw, ChecksumSep)
	if sumSepIndex == -1 {
		return BaseSentence{}, errors.New("nmea: sentence does not contain checksum separator")
	}
	var (
		fieldsRaw   = raw[startIndex+1 : sumSepIndex]
		fields      = strings.Split(fieldsRaw, FieldSep)
	)
	talker, typ := parsePrefix(fields[0])
	return BaseSentence{
		Talker:   talker,
		Type:     typ,
		Fields:   fields[1:],
		Raw:      raw,
		TagBlock: tagBlock,
	}, nil
}

// parsePrefix takes the first field and splits it into a talker id and data type.
func parsePrefix(s string) (string, string) {
	if strings.HasPrefix(s, "PMTK") {
		return "PMTK", s[4:]
	}
	if strings.HasPrefix(s, "P") {
		return "P", s[1:]
	}
	if len(s) < 2 {
		return s, ""
	}
	return s[:2], s[2:]
}

// Checksum xor all the bytes in a string an return it
// as an uppercase hex string
func Checksum(s string) string {
	var checksum uint8
	for i := 0; i < len(s); i++ {
		checksum ^= s[i]
	}
	return fmt.Sprintf("%02X", checksum)
}

// MustRegisterParser register a custom parser or panic
func MustRegisterParser(sentenceType string, parser ParserFunc) {
	if err := RegisterParser(sentenceType, parser); err != nil {
		panic(err)
	}
}

// RegisterParser register a custom parser
func RegisterParser(sentenceType string, parser ParserFunc) error {
	customParsersMu.Lock()
	defer customParsersMu.Unlock()

	if _, ok := customParsers[sentenceType]; ok {
		return fmt.Errorf("nmea: parser for sentence type '%q' already exists", sentenceType)
	}

	customParsers[sentenceType] = parser
	return nil
}

// Parse parses the given string into the correct sentence type.
func Parse(raw string) (Sentence, error) {
	s, err := parseSentence(raw)
	if err != nil {
		return nil, err
	}

	// Custom parser allow overriding of existing parsers
	if parser, ok := customParsers[s.Type]; ok {
		return parser(s)
	}

	if strings.HasPrefix(s.Raw, SentenceStart) {
		// MTK message types share the same format
		// so we return the same struct for all types.
		switch s.Talker {
		case TypeMTK:
			return newMTK(s)
		}

		switch s.Type {
		case TypeRMC:
			return newRMC(s)
		case TypeGGA:
			return newGGA(s)
		case TypeGSA:
			return newGSA(s)
		case TypeGLL:
			return newGLL(s)
		case TypeVTG:
			return newVTG(s)
		case TypeZDA:
			return newZDA(s)
		case TypePGRME:
			return newPGRME(s)
		case TypeGSV:
			return newGSV(s)
		case TypeHDT:
			return newHDT(s)
		case TypeGNS:
			return newGNS(s)
		case TypeTHS:
			return newTHS(s)
		case TypeWPL:
			return newWPL(s)
		case TypeRTE:
			return newRTE(s)
		case TypeVHW:
			return newVHW(s)
		case TypeDPT:
			return newDPT(s)
		case TypeDBT:
			return newDBT(s)
		case TypeDBS:
			return newDBS(s)
		}
	}
	if strings.HasPrefix(s.Raw, SentenceStartEncapsulated) {
		switch s.Type {
		case TypeVDM, TypeVDO:
			return newVDMVDO(s)
		}
	}
	return nil, errors.New("nmea: sentence prefix not supported")
}
