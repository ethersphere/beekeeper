package bee

type Bees []BeeV2

func (c *CheckCase) Bees() Bees {
	return Bees(c.nodes)
}

type FilterFunc func(b *BeeV2) bool

func (bb Bees) Filter(f FilterFunc) (out Bees) {
	for _, b := range bb {
		if f(&b) {
			out = append(out, b)
		}
	}

	return
}

type ConsumeFunc func(b *BeeV2) error

func (bb Bees) ForEach(c ConsumeFunc) error {
	for _, b := range bb {
		if err := c(&b); err != nil {
			return err
		}
	}

	return nil
}
