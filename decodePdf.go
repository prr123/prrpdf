// program to analyse pdf files in go
// author: prr
// date 29/2/2022
// copyright 2022 prr azul software
//

package main

import (
	"os"
	"fmt"
	"log"
	"strings"
	"pdf/azulpdf/pdfLib"

    util "github.com/prr123/utility/utilLib"
)


func main() {

	numArgs:= len(os.Args)

	if numArgs < 2 {
		fmt.Printf("error - exit: insufficient command line arguments\n")
		fmt.Printf("usage is: parsePdf \"file\" [\\out=] [\\dbg]\n")
		os.Exit(-1)
	}

	parseFilnam :=os.Args[1]

	flags := [] string {"out", "dbg"}

	argmap, err := util.ParseFlagsStart(os.Args, flags,2)
	if err != nil {log.Printf("error ParseFlags: %v\n", err); os.Exit(-1);}

	outFilNam, ok := argmap["out"]
	if !ok {
   		pos := strings.Index(parseFilnam, ".pdf")
    	if pos == -1 {fmt.Printf("error parseFilnam has no pdf extension!\n"); os.Exit(-1);}
		outFilNam = parseFilnam[0:(pos+1)] + "pdfdat"
	}

	outFilNamStr := outFilNam.(string)
fmt.Printf("out file: %s\n",outFilNamStr)

	pdf, err := pdflib.InitPdfLib(parseFilnam)
	if err != nil {log.Fatalf("InitPdfLib: %v\n", err)}

//	err = pdf.ReadPdf(parseFilnam)
//	if err != nil {fmt.Printf("error ReadPdf file: %s! %v\n", parseFilnam, err); os.Exit(-1);}

	err = pdf.DecodePdfToText(outFilNamStr)
	if err != nil {log.Fatalf("DecodePdf: %v\n", err)}

	pdf.PrintPdf()

	log.Println("success DecodePdf!")
}
