// program to parse pdf files
// author: prr
// date: 18/1/2022
// copyright 2022 prr azul software
//
package main

import (
	"os"
	"fmt"
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

	err = pdf.ParsePdf()
	if err != nil {
		fmt.Printf("error ParsePdf: %v\n", err)
		os.Exit(2)
	}


	fmt.Println("success!")
}
