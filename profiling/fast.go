package main

import (
	// "bufio"
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

type User struct {
	Browsers []string
	Email    string
	Name     string
}

func FastSearch(out io.Writer) {
	seenBrowsers := make(map[string]bool)

	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	fmt.Fprintln(out, "found users:")
	for i := 0; scanner.Scan(); i++ {
		line := scanner.Bytes()
		isAndroid := false
		isMSIE := false

		user := User{}
		if err := user.UnmarshalJSON(line); err != nil {
			panic(err)
		}

		for _, browser := range user.Browsers {
			if ok := strings.Contains(browser, "Android"); ok {
				isAndroid = true
				seenBrowsers[browser] = true
			}
			if ok := strings.Contains(browser, "MSIE"); ok {
				isMSIE = true
				seenBrowsers[browser] = true
			}
		}

		if !(isAndroid && isMSIE) {
			continue
		}

		// log.Println("Android and MSIE user:", user["name"], user["email"])
		email := strings.ReplaceAll(user.Email, "@", " [at] ")
		fmt.Fprintf(out, "[%d] %s <%s>\n", i, user.Name, email)
	}
	fmt.Fprintln(out, "\nTotal unique browsers", len(seenBrowsers))
}
