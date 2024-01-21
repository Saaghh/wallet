package service

type Service struct {
	visitHistory map[string]int
}

func New() *Service {
	history := make(map[string]int)

	return &Service{
		visitHistory: history,
	}
}

func (s Service) SaveVisit(addr string) {
	s.visitHistory[addr]++
}

func (s Service) GetVisitHistory() map[string]int {
	return s.visitHistory
}
