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
	"prrpdf/pdfLib"
	util "prrpdf/utilLib"
)


func main() {

	numArgs:= len(os.Args)

	if numArgs < 2 {
		fmt.Printf("error - exit: insufficient command line arguments\n")
		fmt.Printf("usage is: parsePdf \"file\"\n")
	}

	parseFilnam :=os.Args[1]

	flags := [] string {"out", "dbg"}

	argmap, err := util.ParseFlagsStart(os.Args, flags,2)
	if err != nil {fmt.Printf("error ParseFlags: %v\n", err); os.Exit(-1);}

	outFilNam, ok := argmap["out"]
	if !ok {
   		pos := strings.Index(parseFilnam, ".pdf")
    	if pos == -1 {fmt.Printf("error parseFilnam has no pdf extension!\n"); os.Exit(-1);}
		outFilNam = parseFilnam[0:(pos+1)] + "pdfdat"
	}

	outFilNamStr := outFilNam.(string)
fmt.Printf("out file: %s\n",outFilNamStr)

	pdf := pdflib.Init()

	err = pdf.ReadPdf(parseFilnam)
	if err != nil {fmt.Printf("error ReadPdf file: %s! %v\n", parseFilnam, err); os.Exit(-1);}

	pdf.PrintPdf()

	err = pdf.CheckPdf(outFilNamStr)
	if err != nil {fmt.Printf("error CheckPdf: %v\n", err); os.Exit(-1);}

	fmt.Println("success checkPdf!")
}
