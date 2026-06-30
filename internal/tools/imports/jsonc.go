package imports

// stripJSONComments removes // line and /* */ block comments so JSONC-style
// tsconfig files parse with encoding/json. String literals are respected.
func stripJSONComments(data []byte) []byte {
	s := &jsoncStripper{data: data}
	for s.i = 0; s.i < len(data); s.i++ {
		s.step()
	}
	return s.out
}

// jsoncStripper is a small state machine over the raw bytes of a JSONC file.
type jsoncStripper struct {
	data                      []byte
	out                       []byte
	i                         int
	inString, inLine, inBlock bool
}

func (s *jsoncStripper) step() {
	c := s.data[s.i]
	switch {
	case s.inLine:
		if c == '\n' {
			s.inLine = false
			s.out = append(s.out, c)
		}
	case s.inBlock:
		if s.peek('*', '/') {
			s.inBlock = false
			s.i++
		}
	case s.inString:
		s.stringByte(c)
	case c == '"':
		s.inString = true
		s.out = append(s.out, c)
	case s.peek('/', '/'):
		s.inLine = true
		s.i++
	case s.peek('/', '*'):
		s.inBlock = true
		s.i++
	default:
		s.out = append(s.out, c)
	}
}

// peek reports whether the current and next byte are a and b.
func (s *jsoncStripper) peek(a, b byte) bool {
	return s.data[s.i] == a && s.i+1 < len(s.data) && s.data[s.i+1] == b
}

// stringByte copies a byte inside a string literal, handling escapes and the
// closing quote.
func (s *jsoncStripper) stringByte(c byte) {
	s.out = append(s.out, c)
	switch {
	case c == '\\' && s.i+1 < len(s.data):
		s.out = append(s.out, s.data[s.i+1])
		s.i++
	case c == '"':
		s.inString = false
	}
}
