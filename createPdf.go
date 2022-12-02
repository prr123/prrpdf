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
		fmt.Printf("usage is: createPdf \"file\" [\\out=] [\\dbg]\n")
		os.Exit(-1)
	}

	createFilnam :=os.Args[1]

	pos := strings.Index(createFilnam, ".pdf")
   	if pos == -1 {fmt.Printf("error createFilnam has no pdf extension!\n"); os.Exit(-1);}

	flags := [] string {"out", "dbg"}

	argmap, err := util.ParseFlagsStart(os.Args, flags,2)
	if err != nil {fmt.Printf("error ParseFlags: %v\n", err); os.Exit(-1);}

	outFilNam, ok := argmap["out"]
	if !ok {
//		outFilNam = parseFilnam[0:(pos+1)] + "pdfdat"
	}

	outFilNamStr := outFilNam.(string)
fmt.Printf("out file: %s\n",outFilNamStr)

	pdf := pdflib.Init()

//	err = pdf.ReadPdf(parseFilnam)
//	if err != nil {fmt.Printf("error ReadPdf file: %s! %v\n", parseFilnam, err); os.Exit(-1);}

	err = pdf.CreatePdf(createFilnam)
	if err != nil {fmt.Printf("error CreatePdf: %v\n", err); os.Exit(-1);}

//	pdf.PrintPdf()

	fmt.Println("success createPdf!")
}
