package diff_test

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"kr.dev/diff"
)

func TestStringTypeMatch(t *testing.T) {
	type B []byte
	testStringDiff(t, linesMyers, []byte(linesA), []byte(linesB))
	testStringDiff(t, linesMyers, B(linesA), B(linesB))
	testStringDiff(t, runesRunes, []rune(runesA), []rune(runesB))
}

func TestTextLines(t *testing.T) {
	testStringDiff(t, linesMyers, linesA, linesB)
}

func TestTextWSOnly(t *testing.T) {
	testStringDiff(t, wsonlyMyers, wsonlyA, wsonlyB)
}

func TestTextWords(t *testing.T) {
	testStringDiff(t, wordsMyers, wordsA, wordsB)
}

func TestTextShort(t *testing.T) {
	testStringDiff(t, shortDiff, shortA, shortB)
}

func TestTextRunes(t *testing.T) {
	testStringDiff(t, runesMyers, runesA, runesB)
}

func TestLogMyers(t *testing.T) {
	var buf bytes.Buffer
	l := log.New(&buf, "", log.Lshortfile)
	type Y struct{ S string }
	diff.Log(Y{"a\nb"}, Y{"a\nc"}, diff.Logger(l))
	got := strings.TrimSpace(buf.String())
	want := "text_test.go:"
	if !strings.HasPrefix(got, want) {
		t.Errorf("diff.Log() = %q, want prefix %q", got, want)
	}
	want = "\n--- a\n+++ b\n@@ -1,2 +1,2 @@\n a\n-b\n+c"
	if !strings.HasSuffix(got, want) {
		t.Errorf("diff.Log() = %q, want suffix %q", got, want)
	}
}

func testStringDiff(t *testing.T, want string, a, b any) {
	t.Helper()
	var got string
	gotp := (*stringPrinter)(&got)
	diff.Each(gotp.Printf, a, b)
	if got != want {
		t.Errorf("bad diff")
		t.Logf("got:\n%s", got)
		t.Logf("want:\n%s", want)
	}
}

const linesA = `
public class File1 {

  public int add (int a, int b)
  {
    log();
    return a + b;
  }

  public int sub (int a, int b)
  {
    if (a == b)
    {
        return 0;
    }
    log();
    return a - b;
    // TOOD: JIRA1234
  }

}
`

const linesB = `
public class File1 {

  public int sub (int a, int b)
  {
    // TOOD: JIRA1234
    if ( isNull(a, b) )
    {
        return null
    }
    log();
    return a - b;
  }

  public int mul (int a, int b)
  {
    if ( isNull(a, b) )
    {
        return null;
    }
    log();
    return a * b;
  }

}
`

const linesMyers = `--- a
+++ b
@@ -1,21 +1,25 @@
 
 public class File1 {
 
-  public int add (int a, int b)
+  public int sub (int a, int b)
   {
+    // TOOD: JIRA1234
+    if ( isNull(a, b) )
+    {
+        return null
+    }
     log();
-    return a + b;
+    return a - b;
   }
 
-  public int sub (int a, int b)
+  public int mul (int a, int b)
   {
-    if (a == b)
+    if ( isNull(a, b) )
     {
-        return 0;
+        return null;
     }
     log();
-    return a - b;
-    // TOOD: JIRA1234
+    return a * b;
   }
 
 }

`

const wordsA = `The brown fox jumped over the lazy dog's tail.`
const wordsB = `The quick brown fox jumps over lazy dog.`

const wordsMyers = `string[4:4]: "" != "quick "
string[14:21]: "jumped " != "jumps "
string[26:30]: "the " != ""
string[35:46]: "dog's tail." != "dog."
`

const (
	shortA    = `Wherever you go,`
	shortB    = `there you are.`
	shortDiff = `"Wherever you go," != "there you are."` + "\n"
)

const (
	runesA = "cqf8vNoIhmGwXgajTst/OKqkm9M"
	runesB = "cqf8vNoInmGwXgojTst/OKqkm9M="

	runesMyers = `string[8:9]: "h" != "n"
string[14:15]: "a" != "o"
string[27:27]: "" != "="
`

	runesRunes = `[]int32[8]: 104 != 110
[]int32[14]: 97 != 111
[]int32[27:27]: {} != {61}
`
)

const (
	wsonlyA     = "x\n    y\nz"
	wsonlyB     = "x\n\ty\nz"
	wsonlyMyers = "--- a\n" +
		"+++ b\n" +
		"@@ -1,3 +1,3 @@\n" +
		" x\n" +
		"-\u00b7\u00b7\u00b7\u00b7y\n" +
		"+ \u2192 y\n" +
		" z\n\n"
)
