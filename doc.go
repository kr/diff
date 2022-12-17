/*
Package diff finds the differences between a pair of Go
values.

The Test, Log, and Each functions all traverse their two
arguments, a and b, in parallel, looking for
differences. Each difference is emitted to the given
testing output function, logger, or callback function.

Here are some common usage examples:

	diff.Test(t, t.Errorf, got, want)
	diff.Test(t, t.Fatalf, got, want)
	diff.Test(t, t.Logf, got, want)

	diff.Log(a, b)
	diff.Log(a, b, diff.Logger(log.New(...)))

	diff.Each(fmt.Printf, a, b)

Use Option values to change how it works if the default
behavior isn't what you need.
*/
package diff
