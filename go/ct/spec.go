package ct

type Specification interface {
	GetRules() []Rule
	GetRulesFor(State) []Rule
}

type specification struct {
	rules []Rule
}

func NewSpecification(rules ...Rule) Specification {
	return &specification{rules}
}

func (s *specification) GetRules() []Rule {
	return s.rules
}

func (s *specification) GetRulesFor(state State) []Rule {
	res := []Rule{}
	for _, rule := range s.rules {
		if rule.Condition.Check(state) {
			res = append(res, rule)
		}
	}
	return res
}
