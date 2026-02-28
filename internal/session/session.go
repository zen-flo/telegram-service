package session

type Session struct {
	id string
}

func New(id string) *Session {
	return &Session{
		id: id,
	}
}

func (s *Session) ID() string {
	return s.id
}
