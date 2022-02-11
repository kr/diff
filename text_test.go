package diff_test

import (
	"fmt"
	"testing"

	"kr.dev/diff"
)

func TestStringTypeMatch(t *testing.T) {
	type B []byte
	testStringDiff(t, linesMyers, []byte(linesA), []byte(linesB))
	testStringDiff(t, linesMyers, B(linesA), B(linesB))
	testStringDiff(t, "{len 233} != {len 300}\n", []rune(linesA), []rune(linesB))
}

func TestTextLines(t *testing.T) {
	testStringDiff(t, linesMyers, linesA, linesB)
}

func TestTextWords(t *testing.T) {
	testStringDiff(t, wordsMyers, wordsA, wordsB)
}

func TestTextShort(t *testing.T) {
	testStringDiff(t, shortDiff, shortA, shortB)
}

func testStringDiff(t *testing.T, want string, a, b any) {
	t.Helper()
	var got string
	f := func(format string, arg ...any) {
		got = fmt.Sprintf(format, arg...)
	}
	diff.Each(f, a, b)
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

const wordsA = `The quick brown fox jumped over the lazy dog's tail.`
const wordsB = `The quick brown fox jumps over the lazy dog.`

const wordsMyers = `--- a
+++ b
@@ -2,9 +2,8 @@
 quick 
 brown 
 fox 
-jumped 
+jumps 
 over 
 the 
 lazy 
-dog's 
-tail.
+dog.

`

const (
	shortA    = `Wherever you go,`
	shortB    = `there you are.`
	shortDiff = `"Wherever you go," != "there you are."` + "\n"
)
