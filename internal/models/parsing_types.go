package models

type ParsingType string

const (
	Instagram ParsingType = "instagram"
)

var parsingTypes = map[ParsingType]struct{}{
	Instagram: struct{}{},
}

func IsValidParsingType(s string) bool {
	_, ok := parsingTypes[ParsingType(s)]
	return ok
}
