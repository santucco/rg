package main

import (
	"bufio"
	"flag"
	"fmt"
	"bitbucket.org/santucco/hoc"
	"os"
	"strings"
	"unicode/utf8"
)

var Usage = func() {
	fmt.Fprintf(os.Stderr, "Report generator\n")
	fmt.Fprintf(os.Stderr, "Copyright (C) Alexander Sychev, 2010, 2012\n")
	fmt.Fprintf(os.Stderr, "Usage of %s: -o <outfile> <filename1> [filenameN]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\t-o <outfile> - output file for storing state\n")
	fmt.Fprintf(os.Stderr, "\t<filenameN> - input files\n")
}
var begin, end string
var h *hoc.Hoc = new(hoc.Hoc)

func main() {
	var outfile string
	flag.StringVar(&begin, "b", "<rg>", "begin mark")
	flag.StringVar(&end, "e", "</rg>", "end mark")
	flag.StringVar(&outfile, "o", "", "output filename")
	flag.Parse()

	h.IndentSym = "\t"
	h.NewLine = true
	defer recov()

	if flag.NArg() == 0 {
		Usage()
		return
	}
	for i := 0; i < flag.NArg(); i++ {
		inf, err := os.Open(flag.Arg(i))
		if err != nil {
			panic(err)
		}
		process(inf, os.Stdout)
	}
	if len(outfile) == 0 {
		return
	}
	outf, err := os.OpenFile(outfile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		panic(err)
	}
	defer outf.Close()
	rf, wf, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	outf.WriteString(begin + "\n")
	wf.WriteString(begin + "print" + end)
	wf.Close()
	process(rf, outf)
	outf.WriteString(end + "\n")

}

func process(inf *os.File, outf *os.File) {
	bin := bufio.NewReader(inf)
	if bin == nil {
		panic("can't make reader")
	}

	code := false
	var in, out chan string
	for bin != nil {
		s, err := bin.ReadString('\n')
		if err != nil && len(s) == 0 {
			bin = nil
			break
		}

		for len(s) > 0 {
			switch code {
			case false:
				if i := strings.Index(s, begin); i != -1 {
					in = make(chan string)
					out = make(chan string)
					go func() {
						defer recov()
						h.Process(in, out)
					}()
					code = true
					c, err := outf.WriteString(s[:i])
					if len(s[:i]) > c {
						panic(err)
					}
					s = s[i+len(begin):]
				} else {
					c, err := outf.WriteString(s)
					if len(s) > c {
						panic(err)
					}
					s = ""
				}
			case true:
				if i := strings.Index(s, end); i != -1 {
					code = false
					in <- s[:i]
					close(in)
					for true {
						s, ok := <-out
						if !ok {
							break
						}
						if len(s) == 0 {
							continue
						}
						outf.WriteString(s)
					}
					s = s[i+len(end):]
				} else {
					in <- s
					s = ""
				}
			}
		}
	}
}

func recov() {
	if x := recover(); x != nil {
		switch x.(type) {
		case hoc.HocError:
			var err string
			e := x.(hoc.HocError)
			if e.Lineno > 0 {
				f := fmt.Sprintf("\n%%s\n%%%ds",
					utf8.RuneCountInString(e.Line[0:e.Linepos]))
				err = fmt.Sprintf(f, strings.TrimRight(e.Line, "\n"), "^")
			}
			fmt.Fprintf(os.Stderr, "\npanic: %v %s\n", e.Error, err)
		default:
			fmt.Fprintf(os.Stderr, "panic: %v\n", x)
		}
	}
}
