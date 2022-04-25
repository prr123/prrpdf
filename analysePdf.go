// program to analyse pdf files in go
// author: prr
// date 29/2/2022
// copyright 2022 prr azul software
//

package main

import (
	"os"
	"fmt"
	"strings"
	"prrpdf/pdflib"
)


func main() {

	numArgs:= len(os.Args)
	fmt.Printf("num Args: %d\n", numArgs)

	if numArgs < 2 {
		fmt.Printf("error - exit: insufficient command line arguments\n")
		fmt.Printf("usage is: parsePdf \"file\"\n")
	}

	parseFilnam :=os.Args[1]

	pdf := pdflib.Init()

	err := pdf.ReadPdf(parseFilnam)
	if err != nil {
		fmt.Printf("error ReadPdf file: %s! %v\n", parseFilnam, err)
		os.Exit(2)
	}

	pdf.PrintPdf()

   	pos := strings.Index(parseFilnam, ".pdf")
    if pos == -1 {
        fmt.Printf("error parseFilnam has no pdf extension!\n")
		os.Exit(2)
    }

	outfilnam := parseFilnam[0:pos]
	err = pdf.AnalysePdf(outfilnam)
	if err != nil {
		fmt.Printf("error ParsePdf: %v\n", err)
		os.Exit(2)
	}


	fmt.Println("success!")
}
