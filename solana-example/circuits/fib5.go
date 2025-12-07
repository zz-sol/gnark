package circuits

import "github.com/consensys/gnark/frontend"

// Fib5 proves that the 5th Fibonacci value equals Out, starting from (0,1).
type Fib5 struct {
	Out frontend.Variable `gnark:",public"`
}

func (c *Fib5) Define(api frontend.API) error {
	a, b := frontend.Variable(0), frontend.Variable(1)
	for i := 0; i < 4; i++ {
		a, b = b, api.Add(a, b)
	}
	api.AssertIsEqual(b, c.Out)
	return nil
}
