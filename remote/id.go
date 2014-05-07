package remote

type ID string

func (id ID) Short() ID {
	shortLen := 12
	if len(id) < shortLen {
		shortLen = len(id)
	}
	return id[:shortLen]
}

func (id ID) String() string {
	return string(id)
}
