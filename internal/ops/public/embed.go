package public

import _ "embed"

//go:embed visitor.html
var VisitorHTML []byte

//go:embed survey.html
var SurveyHTML []byte

//go:embed fault.html
var FaultHTML []byte

//go:embed fault_query.html
var FaultQueryHTML []byte
