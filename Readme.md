# kr.dev/diff

Hello, friend! This is a Go module to print the
differences between any two Go values.

    diff.Test(t, t.Errorf, got, want)

There are a few ways to use this module, but this here ☝️
is probably the most common. In test code, to check
whether the result of your test is what you expected.

# Usage

Test for expected value in a test:

    func TestFoo(t *testing.T) {
        got := ...
        want := ...
        diff.Test(t, t.Errorf, got, want)
    }

Log diffs in production:

    diff.Log(a, b)

Print diffs to stdout:

    diff.Each(fmt.Printf, a, b)

There are also several options to change how it works if
the default behavior isn't what you need. Check out the
godoc at [kr.dev/diff](https://kr.dev/diff).

# Design Philosophy

Here are some general guidelines I try to follow for
this module:

- **make it easy to read for humans** (not computers)
- **use Go-style notation** for familiarity
- **favor being concise over being explicit**
- but **be explicit where necessary** to avoid confusion

These aren't hard rules.

For instance, although it is a goal to make the output
readable to someone who's familiar with Go, I make no
effort to strictly adhere to Go syntax.

## Don't Rely on Code Under Test

We also avoid calling methods on the values being
compared. For instance, package `time` defines a method
`Time.Equal`, that tells whether two `Time` values are
the same instant, regardless of their locations. But we
don't use it or any other `Equal` method. Instead, we
have custom comparison logic (`TimeEqual`) to compare
`Time` values while ignoring their locations.

The reason for this is that you might be trying to test
your `Equal` method! It would be really confusing if
there's a bug in the code you're testing, and that
causes this package to produce incorrect results. You
might end up with a "passing" test because `Equal`
returns true even when the values are different. We want
to reliably show you when the values are different.

So our policy is this module **doesn't call methods on the
values being compared**. Instead, if you need to
customize how comparisons are done, you can install a
transform. See [Custom Comparison](#custom-comparison).
Your transform is free to call methods on the values being
compared, that is up to you; this module will simply not
do so directly. Our hope is that if you're doing it
yourself, it'll be less surprising.

# Custom Comparison

Sometimes you want or need to customize how values of a
given type are compared. Here's how.

Let's say you're testing code that uses temporary files
with randomized names, and your result might contain
values of `*fs.PathError`. You want to check that the
errors are the same, but you need to ignore the file
name. By design, the name is different every time.

    var ignorePath = diff.Transform(func(pe *fs.PathError) any {
        return &fs.PathError{
            Op:   pe.Op,
            Path: "", // ignore pe.Path
            Err:  pe.Err,
        }
    })

    diff.Test(t, t.Errorf, a, b, ignorePath)

In this case, you use the `Transform` option to
change each `*fs.PathError` into a new value, so that
the transformed values are equal as long as `Op` and
`Err` are equal, and unequal otherwise.

There are also a couple of predefined transforms
exported by this package. Their definitions are visible
in the godoc at [kr.dev/diff](https://kr.dev/diff).

Side note. Why doesn't the option look like this?

    diff.CustomCompare(equal func(a, b T) bool) Option

That's because, under the hood, we don't always directly
compare two values for equality. Sometimes we also hash
the values and compare their hashes. If you wanted to
define a custom boolean equality function, you'd also
have to provide a custom hashing function. But with a
transform, this module can do both the hashing and
comparison for you, and you only need to write one
relatively easy function.

# Custom Formatting

This package tries to provide readable and useful output
out of the box. But sometimes you can do a lot better by
tailoring the output to a specific type. In that case,
you can use the `Format` option to define a custom
format function.

    var fmtFoo = diff.Format(func(a, b Foo) string {
        // TODO(kr): I need a good example to put here.
        // For now, look at diff.TimeDelta in the godoc.
    })

    diff.Test(t, t.Errorf, a, b, fmtFoo)

Your formatter takes two values of the given type and
returns a description of the difference between them. It
only gets called when the values are already determined
to be unequal, so you don't have to compare them. You
can assume they are different. (If you need to customize
how values are compared, see [Custom
Comparison](#custom-comparison).)

There are also a couple of predefined custom formats
exported by this package. Their definitions are visible
in the godoc at [kr.dev/diff](https://kr.dev/diff).

# Compatibility

The output of this package is mainly meant for
humans to read, and it's not a goal to be easy for
computers to parse. We will occasionally change the
format to try to make it better. So please keep this in
mind if you want to make a tool that consumes the output
of this module; it might break!

On top of that, this is still a v0 module, so we might
*also* change the API in a way that breaks.

# Roadmap

No promises here, but this is what I intend to work on:

- [x] example tests
- [ ] fuzz tests
- [x] full output mode
- [x] sort map keys when possible
- [x] detect cycles when formatting full output
- [ ] histogram/myers diff for lists and slices
- [x] special handling for text (string and []byte)
- [ ] special handling for whitespace-only diffs
- [ ] special handling for binary (string and []byte)
- [ ] format single value API (package, maybe module?)
- [ ] make depth limit configuable (as "precision")

# Feedback

If you find bugs or want more features or have design
feedback about the interface, please file issues or pull
requests! Thank you!
