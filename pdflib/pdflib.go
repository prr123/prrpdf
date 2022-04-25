// library to analyse pdf documents
// author: prr
// created: 18/2/2022
//
// 21/2/2022
// added zlib
//

package pdflib

import (
	"os"
	"fmt"
	"strconv"
	"strings"
	"bytes"
	"io"
	"compress/zlib"
)

type InfoPdf struct {
	fil *os.File
	filSize int64
	filNam string
	numObj int
	trail_root int
	trail_size int
	objList *[]docObj
	root int
	pageCount int
	pglist []int
}

type docObj struct {
	objId int
	objTyp int
	parent int
	st int
	end int
}

func Init()(info *InfoPdf) {
	var pdf InfoPdf
	return &pdf
}

func (pdf *InfoPdf) ReadPdf(parseFilnam string) (err error) {

   	parseFil, err := os.Open(parseFilnam)
    if err != nil {
        return fmt.Errorf("error opening file \"%s\"! %v\n", parseFilnam, err)
    }

    filinfo, err := parseFil.Stat()
    if err != nil {
	    return fmt.Errorf("error opening file \"%s\"! %v\n", parseFilnam, err)
    }


	pdf.fil = parseFil
	pdf.filSize = filinfo.Size()
	pdf.filNam = parseFilnam
	return nil
}

func (pdf *InfoPdf) PrintPdf(){

    fmt.Printf("File Size %d \n", pdf.filSize)

}

func (pdf *InfoPdf) ParsePdf()(err error) {
//	var outstr string

	buf := make([]byte,pdf.filSize)

	_, err = (pdf.fil).Read(buf)
	if err != nil {
		return fmt.Errorf("error ParsePdf:: Read: %v", err)
	}

	//read top line
	end_fl :=0
	for i:=0; i<len(buf); i++ {
		if (buf[i] == '\r') || (buf[i] == '\n') {
			end_fl = i
			break
		}
	}
	if end_fl ==0 {
		return fmt.Errorf("error ParsePdf: invalid first line!")
	}

	fmt.Printf("first line: %s\n", string(buf[:end_fl]))
	if string(buf[:5]) != "%PDF-" {
		return fmt.Errorf("error ParsePdf: begin %s not \"%%PDF-\"!", string(buf[:5]))
	}
	fmt.Printf("pdf version: %s\n",string(buf[5:end_fl]))
	// second line
	end_sl :=0
	for i:=end_fl+1; i<len(buf); i++ {
		if (buf[i] == '\r') || (buf[i] == '\n') {
			end_sl = i
			break
		}
	}
	if end_sl ==0 {
		return fmt.Errorf("error ParsePdf: invalid second line!")
	}
	fmt.Printf("second line: %s\n", string(buf[end_fl+1:end_sl]))

	// last line
	// first get rid of empty lines at the end
	ll_end := len(buf) -1

	for i:=ll_end; i>0; i-- {
		if (buf[i] != '\r') && (buf[i] != '\n') {
			ll_end = i
			break;
		}
	}

	// now we have the real last line which should contain %%EOF
	start_ll :=0
	for i:=ll_end; i>0; i-- {
		if (buf[i] == '\r') || (buf[i] == '\n') {
			start_ll = i+1
			break
		}
	}

	if start_ll ==0 {
		return fmt.Errorf("error ParsePdf: invalid last line!")
	}

	llstr := string(buf[start_ll:ll_end+1])
	fmt.Printf("last line: %s\n", llstr)
	if llstr != "%%EOF" {
		return fmt.Errorf("error ParsePdf: end %s is not \"%%EOF\"!", llstr)
	}

	//next we need to check second last line which should contain a int number
	sl_end:=start_ll-2
	if (buf[sl_end] == '\r') || (buf[sl_end] == '\n') {
		sl_end--
	}
	startxref_end :=0

	for i:=sl_end; i>0; i-- {
		if (buf[i] == '\r') || (buf[i] == '\n') {
			startxref_end = i
			break
		}
	}
	if startxref_end ==0 {
		return fmt.Errorf("error ParsePdf: invalid second last line!")
	}
	slstr := string(buf[startxref_end+1:sl_end+1])

	xref, err := strconv.Atoi(slstr)
	if err != nil {
		return fmt.Errorf("error ParsePdf: second last line not an int: %s",slstr)
	}
	fmt.Printf("second last line: %d\n", xref)

	//third last line
	// the third last line should have the word "startxref"
	tl_end := startxref_end -1
	// if the string ends with two chars /r + /n instead of one char
	if (buf[tl_end] == '\r') || (buf[tl_end] == '\n') {
		tl_end--
	}
	startxref_start :=0

	for i:=tl_end; i>0; i-- {
		if (buf[i] == '\r') || (buf[i] == '\n') {
			startxref_start = i+1
			break
		}
	}
	if startxref_end ==0 {
		return fmt.Errorf("error ParsePdf: invalid third last line!")
	}
	tlstr := string(buf[startxref_start:tl_end+1])
	fmt.Printf("third last line: %s\n", tlstr)
	if tlstr != "startxref" {
		return fmt.Errorf("error ParsePdf: third line from end %s is not \"startxref\"!", tlstr)
	}

	// find trailer
	fmt.Printf("parsing trailer!\n")
	db_end := 0
	for i:=startxref_start; i> 0; i-- {
		if buf[i] == '>' {
			if buf[i-1] == '>' {
				db_end = i
				break
			}
		}
	}

	if db_end == 0 {
		return fmt.Errorf("error ParsePdf: cannot find closing angular bracket for trailer!")
	}
	fmt.Printf("found closing angular brackets!\n")

	db_start := 0
	for i:=db_end; i> 0; i-- {
		if buf[i] == '<' {
			if buf[i-1] == '<' {
				db_start = i
				break
			}
		}
	}
	if db_start == 0 {
		return fmt.Errorf("error ParsePdf: cannot find opening angular bracket for trailer!")
	}
	fmt.Printf("found opening angular brackets! %s %d\n", string(buf[db_start-1: db_start+1]), db_start-1)

	trailer_start :=0

	trailer_end :=0
	for i:=db_start-2; i>0; i-- {
		if (buf[i] != '\r') || (buf[i] != '\n') {
			trailer_end = i
			break
		}
	}

	if trailer_end ==0 {
		return fmt.Errorf("error ParsePdf: trailer end line!")
	}

	for i:=trailer_end -1 ; i>0; i-- {
		if (buf[i] == '\r') || (buf[i] == '\n') {
			trailer_start = i+1
			break
		}
	}
	if trailer_start ==0 {
		return fmt.Errorf("error ParsePdf: trailer start line!")
	}

//	fmt.Printf("trailer start: %d end: %d\n", trailer_start, trailer_end)
	trailerStr := string(buf[trailer_start:trailer_end])
	fmt.Printf("trailer line: %s\n", trailerStr)
	if trailerStr != "trailer" {
		return fmt.Errorf("error ParsePdf: trailer key word %s is not \"trailer\"!", trailerStr)
	}

	// find xref
	tend := trailer_start -1
	icount := 0
	tline := ""
	tstart :=0
	for i:=trailer_start -5 ; i>0; i-- {
		if (buf[i] == '\r') || (buf[i] == '\n') {
			tstart = i+1
			tline = string(buf[i+1:tend])
			tend = i
			icount++
			fmt.Printf("line %d: str: %s\n", icount, tline)
		}
		if tline == "xref" {break}
//		if icount> 15 {break}
	}
	if trailer_start ==0 {
		return fmt.Errorf("error ParsePdf: trailer start line!")
	}

	// objects

//	docObjList := make([]docObj, 20)
	obj_end := tstart
	for i:= obj_end; i>end_sl; i-- {
		if string(buf[i:i+7]) == "endobj" {
			obj_end = i+7
			break
		}
	}

fmt.Printf("endobj %d %d %s\n", end_sl, obj_end, string(buf[obj_end-7: obj_end]))

	obj_start := 0

	for i:= obj_end-10 ; i>end_sl; i-- {
//		fmt.Printf("test:  %s\n", string(buf[i:(i+4)]))
		if string(buf[i:(i+4)]) == " obj" {
			obj_start = i
			break
		}
	}

	if obj_start == 0 {return fmt.Errorf("error pdfParse:: cannot find start for obj")}
	obj_start -=3
	fmt.Printf("obj: %s %d %d\n", string(buf[obj_start: obj_end]), obj_start, obj_end)

	num_end:= 0
	for i:= obj_start; i< obj_start +3; i++ {
		if buf[i] == ' ' {
			num_end = i
		}
	}
	numstr := string(buf[obj_start: num_end])
	objnum, err := strconv.Atoi(numstr)
	if err != nil {
//		outstr = fmt.Sprintf("***error*** cannot convert object number %s to int in xref!\n", numstr)
//		outfil.WriteString(outstr)
		return fmt.Errorf("error AnalysePdf:: cannot convert object number to int in xref!")
	}
	// todo
	fmt.Printf("obj number: %s %d\n", numstr, objnum)

	objbuf := buf[end_sl:tend]
	fmt.Printf("objbuf %d %d length %d\n", end_sl, tend, tend-end_sl)
	objList, err :=pdf.GetPdfObjList(&objbuf)
	fmt.Printf("objlist: %d\n", len(*objList))
    (pdf.fil).Close()
	return nil
}

func (pdf *InfoPdf) GetPdfObjList(buf *[]byte)(objList *[]docObj, err error) {

	fmt.Println("****************GetPdfObjList************")
	bufLen := len(*buf)
	fmt.Printf("GetPdfObjList: %d\n", bufLen)
	fmt.Println("pdf Objects: ", pdf.numObj)
	objList = pdf.objList
	for i:= 1; i< pdf.numObj; i++ {
		fmt.Printf("obj: %d start: %d end %d\n",i,(*objList)[i].st, (*objList)[i].end)
	}
	fmt.Println("****************GetPdfObjList************")

	return objList, err
}

func (pdf *InfoPdf) AnalysePdf(outfilnam string)(err error) {

	if len(outfilnam) < 2 {return fmt.Errorf("error AnalysePdf:: filnam %s too short!",outfilnam)}

	outfilbuf := []byte(outfilnam)
	for i:=0; i<len(outfilnam); i++ {
		if outfilbuf[i] == '.' {
			return fmt.Errorf("error AnalysePdf:: filnam %s has extension!",outfilnam)
		}
	}

	fullOutFilNam := outfilnam + ".pdfdat"
	outfil, err := os.Create(fullOutFilNam)
	if err != nil {
		return fmt.Errorf("error AnalysePdf:: cannot open file: %v", err)
	}
	defer outfil.Close()

	if pdf.fil == nil {return fmt.Errorf("error AnalysePdf:: pdf file has not been read!")}
	if pdf.filSize < 1 {return fmt.Errorf("error AnalysePdf:: pdf file is too small!")}

	outstr := fmt.Sprintf("pdf file:      %s\n", pdf.filNam)
	outstr += fmt.Sprintf("pdf file size: %d\n", pdf.filSize)
	outfil.WriteString(outstr)

	buf := make([]byte,pdf.filSize)

	_, err = (pdf.fil).Read(buf)
	if err != nil {
		outfil.WriteString("***error*** cannot read pdf file!\n")
		return fmt.Errorf("error AnalysePdf:: cannot read pdf file: %v", err)
	}

	outfil.WriteString("***header***\n")

	//read top line
	end_fl :=0
	for i:=0; i<len(buf); i++ {
		if (buf[i] == '\r') || (buf[i] == '\n') {
			end_fl = i
			break
		}
	}
	if end_fl ==0 {
		outfil.WriteString("***error*** invalid first line!\n")
		return fmt.Errorf("error AnalysePdf: invalid first line!")
	}

	fmt.Printf("first line: %s\n", string(buf[:end_fl]))

	if string(buf[:5]) != "%PDF-" {
		outstr = fmt.Sprintf("***error*** first line %s does not begin with \"%%PDF-\"!\n", string(buf[:5]))
		outfil.WriteString(outstr)
		return fmt.Errorf("error AnalysePdf: begin %s not \"%%PDF-\"!", string(buf[:5]))
	}

	if string(buf[5:7]) != "1." {
		outstr = fmt.Sprintf("***error*** pdf version needs to start with 1. not %s!\n",string(buf[5:7]))
		outfil.WriteString(outstr)
		return fmt.Errorf("error AnalysePdf: invalid pdf major version!", string(buf[5:7]))
	}
	fmt.Printf("major version: %s, minor: %s\n", string(buf[5:6]),string(buf[7:end_fl]))
	min_ver, err := strconv.Atoi(string(buf[7:end_fl]))
	if err !=nil {
		outstr = fmt.Sprintf("***error*** pdf minor version is not numeric! %v\n", err)
		outfil.WriteString(outstr)
		return fmt.Errorf("error AnalysePdf pdf minor version is not numeric! %v\n", err)
	}
	if (min_ver < 1) || (min_ver > 7) {
		outstr = fmt.Sprintf("***error*** pdf minor version %d is not valid!\n", min_ver)
		outfil.WriteString(outstr)
		return fmt.Errorf("error AnalysePdf: pdf minor version %d is not valid!\n", min_ver)
	}
	outstr = fmt.Sprintf("pdf version: %s is valid!\n",string(buf[5:end_fl]))
	outfil.WriteString(outstr)
	// second line
	end_sl :=0
	for i:=end_fl+1; i<len(buf); i++ {
		if (buf[i] == '\r') || (buf[i] == '\n') {
			end_sl = i
			break
		}
	}
	if end_sl ==0 {
		outstr = fmt.Sprintf("***error*** second line no EOL!\n")
		outfil.WriteString(outstr)
		return fmt.Errorf("error AnalysePdf: invalid second line!")
	}

	if end_sl-(end_fl+2) != 4 {
		outstr = fmt.Sprintf("***error*** second line length not 4, but %d!\n", end_sl-(end_fl+2))
		outfil.WriteString(outstr)
		return fmt.Errorf("error AnalysePdf: second line length not 4, but %d!", end_sl -(end_fl+2))
	}
	icount:=1
	sl_bin := true
	for i:=end_fl+2; i< end_sl; i++{
		if int(buf[i]) < 121 {
			sl_bin = false
			break
		}
//		fmt.Printf("byte [%d]: %d\n", icount, int(buf[i]))
		icount++
	}
	if sl_bin {
		outstr = fmt.Sprintf("pdf doc contains binary values!\n")
	} else {
		outstr = fmt.Sprintf("pdf doc contains no binary values!\n")
	}
	outfil.WriteString(outstr)

	fmt.Printf("second line: %s\n", string(buf[end_fl+1:end_sl]))

	// last line
	// first get rid of empty lines at the end
	outfil.WriteString("***Ending***\n")

	ll_end := len(buf) -1

	for i:=ll_end; i>0; i-- {
		if (buf[i] != '\r') && (buf[i] != '\n') {
			ll_end = i
			break;
		}
	}

	// now we have the real last line which should contain %%EOF
	start_ll :=0
	for i:=ll_end; i>0; i-- {
		if (buf[i] == '\r') || (buf[i] == '\n') {
			start_ll = i+1
			break
		}
	}

	if start_ll ==0 {
		outstr = fmt.Sprintf("***error*** last line has no EOL!")
		outfil.WriteString(outstr)
		return fmt.Errorf("error AnalysePdf: invalid last line!")
	}

	llstr := string(buf[start_ll:ll_end+1])
	fmt.Printf("last line: %s\n", llstr)
	if llstr != "%%EOF" {
		outstr = fmt.Sprintf("***error*** last string %s is not \"%%EOF\"!", llstr)
		outfil.WriteString(outstr)
		return fmt.Errorf("error AnalysePdf: last string %s is not \"%%EOF\"!", llstr)
	}

	//next we need to check second last line which should contain a int number
	sl_end:=start_ll-2
	if (buf[sl_end] == '\r') || (buf[sl_end] == '\n') {
		sl_end--
	}
	startxref_end :=0

	for i:=sl_end; i>0; i-- {
		if (buf[i] == '\r') || (buf[i] == '\n') {
			startxref_end = i
			break
		}
	}
	if startxref_end ==0 {
		outstr = fmt.Sprintf("***error*** second last line has no EOL!")
		outfil.WriteString(outstr)
		return fmt.Errorf("error AnalysePdf: invalid second last line!")
	}
	slstr := string(buf[startxref_end+1:sl_end+1])

	xref_ptr, err := strconv.Atoi(slstr)
	if err != nil {
		outstr = fmt.Sprintf("***error*** second last line %s does not contain an int! %v", slstr, err)
		outfil.WriteString(outstr)
		return fmt.Errorf("error AnalysePdf: second last line not an int: %s",slstr)
	}
	outstr = fmt.Sprintf("xref offset: %d\n", xref_ptr)
	outfil.WriteString(outstr)

	fmt.Printf("second last line: %d\n", xref_ptr)
	// check xref
	xref_str := string(buf[xref_ptr: xref_ptr+4])
	if xref_str != "xref" {
		outstr = fmt.Sprintf("***error*** xref pointer points to %s not \"xref\"!\n", xref_str)
		outfil.WriteString(outstr)
		return fmt.Errorf("error AnalysePdf: xref pointer points to %s not \"xref\"!\n", xref_str)
	} else {
		outfil.WriteString("xref points to correct xref location!\n")
		fmt.Printf("xref: %s points to correct xref location!\n", xref_str)
	}

	//third last line
	// the third last line should have the word "startxref"
	tl_end := startxref_end -1
	// if the string ends with two chars /r + /n instead of one char
	if (buf[tl_end] == '\r') || (buf[tl_end] == '\n') {
		tl_end--
	}
	startxref_start :=0

	for i:=tl_end; i>0; i-- {
		if (buf[i] == '\r') || (buf[i] == '\n') {
			startxref_start = i+1
			break
		}
	}
	if startxref_end ==0 {
		outstr = fmt.Sprintf("***error*** third last line has no EOL!")
		outfil.WriteString(outstr)
		return fmt.Errorf("error AnalysePdf: invalid third last line!")
	}
	tlstr := string(buf[startxref_start:tl_end+1])
//	fmt.Printf("third last line: %s\n", tlstr)
	if tlstr != "startxref" {
		outstr = fmt.Sprintf("***error*** third last line %s does not contain \"startxref\"!\n", tlstr)
		outfil.WriteString(outstr)
		return fmt.Errorf("error AnalysePdf: third line from end %s is not \"startxref\"!", tlstr)
	}
	outstr = fmt.Sprintf("Ending parsed successfully %d %d!\n", startxref_start, ll_end)
	outfil.WriteString(outstr)

	// find trailer
	outfil.WriteString("***trailer***\n")
	fmt.Printf("parsing trailer!\n")

	db_end := 0
	for i:=startxref_start; i> 0; i-- {
		if buf[i] == '>' {
			if buf[i-1] == '>' {
				db_end = i
				break
			}
		}
	}

	if db_end == 0 {
		outstr = fmt.Sprintf("***error*** cannot find closing angular brackets for trailer!\n")
		outfil.WriteString(outstr)
		return fmt.Errorf("error AnalysePdf: cannot find closing angular brackets for trailer!")
	}
	fmt.Printf("found closing angular brackets!\n")

	db_start := 0
	for i:=db_end; i> 0; i-- {
		if buf[i] == '<' {
			if buf[i-1] == '<' {
				db_start = i
				break
			}
		}
	}
	if db_start == 0 {
		outstr = fmt.Sprintf("***error*** cannot find opening angular brackets for trailer!\n")
		outfil.WriteString(outstr)
		return fmt.Errorf("error AnalysePdf: cannot find opening angular brackets for trailer!")
	}
	fmt.Printf("found opening angular brackets! %s %d\n", string(buf[db_start-1: db_start+1]), db_start-1)

	trailer_start :=0

	trailer_end :=0
	for i:=db_start-2; i>0; i-- {
		if (buf[i] != '\r') || (buf[i] != '\n') {
			trailer_end = i
			break
		}
	}

	if trailer_end ==0 {
		outstr = fmt.Sprintf("***error*** trailer end line has no EOL!\n")
		outfil.WriteString(outstr)
		return fmt.Errorf("error AnalysePdf: trailer end line!")
	}

	for i:=trailer_end -1 ; i>0; i-- {
		if (buf[i] == '\r') || (buf[i] == '\n') {
			trailer_start = i+1
			break
		}
	}
	if trailer_start ==0 {
		outstr = fmt.Sprintf("***error*** trailer start line has no EOL!\n")
		outfil.WriteString(outstr)
		return fmt.Errorf("error AnalysePdf: trailer start line!")
	}

//	fmt.Printf("trailer start: %d end: %d\n", trailer_start, trailer_end)
	trailerStr := string(buf[trailer_start:trailer_end])
	fmt.Printf("trailer line: %s\n", trailerStr)
	if trailerStr != "trailer" {
		outstr = fmt.Sprintf("***error*** trailer key word %s is not \"trailer\"!\n", trailerStr)
		outfil.WriteString(outstr)
		return fmt.Errorf("error AnalysePdf: trailer key word %s is not \"trailer\"!", trailerStr)
	}

	// need to parse trailer content
	trailCont := string(buf[db_start -1:db_end+1])
	fmt.Printf("trailer content:\n %s\n", trailCont)

	sizeIdx := strings.Index(trailCont,"/Size")
	if sizeIdx < 0 {
		outstr = fmt.Sprintf("***error*** no Size keyword found in trailer!\n")
		outfil.WriteString(outstr)
		return fmt.Errorf("error AnalysePdf:: no Size keyword found in trailer!")
	}
//	fmt.Println("trailer /Size:", trailCont[sizeIdx:])
//	sizeIdx2 := strings.Index(trailCont[sizeIdx:],"/")
	sizeIdx2 := 0
	for i:=0; i< len(trailCont); i++ {
		if (trailCont[i] == '\r') || (trailCont[i] == '\n') {
			sizeIdx2 = i
			break
		}
	}
	if sizeIdx2 < 1 {
		outstr = fmt.Sprintf("***error*** no Size no EOL to Size keyword found in trailer!\n")
		outfil.WriteString(outstr)
		return fmt.Errorf("error AnalysePdf:: no EOL to Size keyword found in trailer!")
	}
	sizStr := trailCont[sizeIdx + len("/Size "):sizeIdx2]
	pdf.numObj, err = strconv.Atoi(sizStr)
	if err != nil {
		outstr = fmt.Sprintf("***error*** cannot convert string %s to obj num in xref! %v\n", sizStr, err)
		outfil.WriteString(outstr)
		return fmt.Errorf("error AnalysePdf:: cannot convert object number in xref! %v", err)
	}
	outstr = fmt.Sprintf("trailer properties:\n")
	outstr += fmt.Sprintf("Size/ Obj numbers: %d\n", pdf.numObj)
	outfil.WriteString(outstr)

	fmt.Println("Size: ",sizStr)

	rootIdx :=strings.Index(trailCont, "/Root")
	if rootIdx < 0 {
		outstr = fmt.Sprintf("***error*** no Root keyword found in trailer!\n")
		outfil.WriteString(outstr)
		return fmt.Errorf("error AnalysePdf:: no Root keyword found in trailer!")
	}

	rootIdx2 := 0
	for i:=rootIdx; i< len(trailCont); i++ {
		if (trailCont[i] == '\r') || (trailCont[i] == '\n') {
			rootIdx2 = i
			break
		}
	}

	if rootIdx2 < 1 {
		outstr = fmt.Sprintf("***error*** no EOL to Root keyword found in trailer!\n")
		outfil.WriteString(outstr)
		return fmt.Errorf("error AnalysePdf:: no EOL to Root keyword found in trailer!")
	}


//	rootIdx2 = strings.Index(trailCont[rooIdx +len("/Root ":]," ")

//	rootStr := trailCont[rootIdx + len("/Root"):rootIdx2]
	rootStr := trailCont[rootIdx:rootIdx2]

	rootFlds := strings.Fields(rootStr)
//	for k:=0; k<len(rootFlds); k++ {
//		fmt.Printf("field %d: %s\n", k, rootFlds[k]) 
//	}
	pdf.root, err = strconv.Atoi(rootFlds[1])
	if err!=nil {
		outstr = fmt.Sprintf("***error*** cannot convert root string %s to obj num in xref! %v\n", rootFlds[1], err)
		outfil.WriteString(outstr)
		return fmt.Errorf("error AnalysePdf:: cannot convert root string %s to object number in xref! %v", rootFlds[1], err)
	}
	outstr = fmt.Sprintf("Root: %d!\n", pdf.root)
	outfil.WriteString(outstr)
	// todo check obj rootFlds[1] is the catalog obj!
	fmt.Println("Root: ",rootFlds[1])

	// find xref
	outfil.WriteString("***xref***\n")
//	tend := trailer_start -1
//	icount = 0
	xref_end :=0
	for i:=xref_ptr ; i<trailer_start; i++ {
		if (buf[i] == '\r') || (buf[i] == '\n') {
			xref_end = i
			break
		}
	}

	if xref_end ==0 {
		outstr = fmt.Sprintf("***error*** no EOL to xref keyword found in xref!\n")
		outfil.WriteString(outstr)
		return fmt.Errorf("error AnalysePdf:: no EOL to xref keyword found in xref!")
	}
	xrefStr := string(buf[xref_ptr:xref_end])
	fmt.Printf("xref: %s\n", xrefStr)

	if xrefStr != "xref" {
		outstr = fmt.Sprintf("***error*** xref keyword %s did not match in xref!\n", xrefStr)
		outfil.WriteString(outstr)
		return fmt.Errorf("error AnalysePdf:: xref keyword did not match in xref!")
	}

	// xref number of objects
	xrefObjend :=0
	for i:=xref_end +1 ; i<trailer_start; i++ {
		if (buf[i] == '\r') || (buf[i] == '\n') {
			xrefObjend = i
			break
		}
	}

	xrefObjStr := string(buf[xref_end+1:xrefObjend])
	fmt.Printf("xref Obj: %s\n", xrefObjStr)

	xrefObjnumStr := string(buf[xref_end+3:xrefObjend])
	xrefObjnum, err := strconv.Atoi(xrefObjnumStr)
	if err != nil {
		outstr = fmt.Sprintf("***error*** cannot convert object number %s in xref!\n", xrefObjnumStr)
		outfil.WriteString(outstr)
		return fmt.Errorf("error AnalysePdf:: cannot convert object number in xref!")
	}
	outstr = fmt.Sprintf("xref Objects: %d!\n", xrefObjnum)
	outfil.WriteString(outstr)
	fmt.Printf("Xref Objects: %d\n", xrefObjnum)

	pdf.numObj = xrefObjnum
	docObjList := make([]docObj, xrefObjnum)
	linst := xrefObjend +1
	linend := 0
//	linStr := ""
	for xrefL:=0; xrefL<xrefObjnum; xrefL++ {
		for i:=linst; i< trailer_start; i++ {
			if (buf[i] == '\r') || (buf[i] == '\n') {
				linend = i
//				linStr = string(buf[linst:linend])
//				fmt.Printf("line %3d: %s\n", xrefL, linStr)
				break
			}
		}
		if linend < linst {
			outstr = fmt.Sprintf("***error*** no EOL found in line %d of xref!\n", xrefL)
			outfil.WriteString(outstr)
			return fmt.Errorf("error AnalysePdf:: no EOL found in line %d of xref!", xrefL)
		}
		objPtrStr := string(buf[linst: linst+10])
//		fmt.Printf("Obj %3d: %s\n", xrefL, objPtrStr)
		objPtr, err := strconv.Atoi(objPtrStr)
		if err != nil {
			outstr = fmt.Sprintf("***error*** cannot convert object number %s in line %d of xref!\n", objPtrStr, xrefL)
			outfil.WriteString(outstr)
			return fmt.Errorf("error AnalysePdf:: cannot convert object number %s in line %d of xref!", objPtrStr, xrefL)
		}
		objLinEnd:=0
		for j:=objPtr; j<objPtr + 20; j++ {
			if (buf[j] == '\r') || (buf[j] == '\n') {
				objLinEnd = j
				break
			}
		}
		objstr := string(buf[objPtr: objLinEnd])
		fmt.Printf("Obj %3d: %6d: %10s\n", xrefL, objPtr, objstr)
		if xrefL > 0 {
			objPtrEnd:=0
			for j:=objPtr; j<trailer_start; j++ {
				if string(buf[j:j+6]) == "endobj" {
					objPtrEnd = j+6
					break
				}
			}
			outstr = fmt.Sprintf("Obj %3d: start: %6d end: %6d\n", xrefL, objPtr, objPtrEnd)
			outfil.WriteString(outstr)
			docObjList[xrefL].st = objPtr
			docObjList[xrefL].end = objPtrEnd
			objstr:= string(buf[objPtr:objPtrEnd])
			if len(objstr) < 80 {fmt.Printf("***\n%s\n***\n",objstr)}
		}
		linst = linend +1
	}
	pdf.objList = &docObjList
/*
	fmt.Printf("\n*********** pdf ***********\n")
	for i:=0; i< pdf.numObj; i++ {
		fmt.Printf("obj: %3d start %6d end: %6d\n", i, (*pdf.objList)[i].st, (*pdf.objList)[i].end)
	}
*/
	fmt.Printf("root: %d\n", pdf.root)

	outfil.WriteString("***key Objects***\n")
	outstr = fmt.Sprintf("root object: %d start: %d\n", pdf.root, (*pdf.objList)[pdf.root-1].st)
	outfil.WriteString(outstr)
	rootStr = string(buf[(*pdf.objList)[pdf.root].st:(*pdf.objList)[pdf.root].end])
	if strings.Contains(rootStr, "Catalog") {
		outfil.WriteString("  -contains the attribute \"Catalog\"!\n")
	} else {
		outfil.WriteString("***error*** root object does not contain the attribute \"Catalog\"!\n")
	}

	if strings.Contains(rootStr, "Pages") {
		outfil.WriteString("  -contains the attribute \"Pages\"!\n")
		pgidx := strings.Index(rootStr, "Pages")
		pgidx2 := strings.Index(rootStr, "R")
		pgStr := rootStr[pgidx:pgidx2]
		pgFlds := strings.Fields(pgStr)
//		fmt.Printf("page str: %s field 1: %s\n",pgStr,pgFlds[1])
		pagesObj, err := strconv.Atoi(pgFlds[1])
		if err != nil {
			outstr = fmt.Sprintf("***error*** cannot convert pages obj id %s!%v\n", pgFlds[1], err)
			outfil.WriteString(outstr)
			return fmt.Errorf("error AnalysePdf:: cannot convert pages obj id %s!%v", pgFlds[1], err)
		}
		pdf.pageCount = pagesObj
		outstr = fmt.Sprintf("  -Pages Obj: %d\n", pdf.pageCount)
		outfil.WriteString(outstr)
	} else {
		outfil.WriteString("***error*** root object does not contain the attribute \"Pages\"!\n")
	}

	outstr = fmt.Sprintf("Pages Object: %d\n", pdf.pageCount)
	outfil.WriteString(outstr)

//	pagesbuf := buf[(*pdf.objList)[pdf.pages].st:(*pdf.objList)[pdf.pages].end]
	pagesStr := string(buf[(*pdf.objList)[pdf.pageCount].st:(*pdf.objList)[pdf.pageCount].end])
	pagesBuf := []byte(pagesStr)
	if strings.Contains(pagesStr, "Count") {
		outfil.WriteString("  -contains the attribute \"Count\"!\n")
		idx1 := strings.Index(pagesStr,"/Count")
		idx2 := 0
		for j:=idx1; j<len(pagesBuf); j++ {
			if (pagesBuf[j] == '\r') || (pagesBuf[j] == '\n') {
				idx2 = j
				break
			}
		}
//		fmt.Printf("count str: %d %d %s\n",idx1, idx2, pagesStr[idx1:idx2])
		pagesFlds := strings.Fields(pagesStr[idx1:idx2])
//		fmt.Printf("count field 1: %s\n",pagesFlds[1])
		pgCount, err := strconv.Atoi(pagesFlds[1])
		if err != nil {
			outstr = fmt.Sprintf("***error*** cannot convert Count %s in Pages to int!%v\n", pagesFlds[1], err)
			outfil.WriteString(outstr)
			return fmt.Errorf("error AnalysePdf:: cannot convert Count %s in Pages to int!%v", pagesFlds[1], err)
		}
		pdf.pageCount = pgCount
	} else {
		outfil.WriteString("***error*** pages object does not contain the attribute \"Count\"!\n")
	}

	if strings.Contains(pagesStr, "Kids") {
		outfil.WriteString("  -contains the attribute \"Kids\"!\n")
		kidx1 := strings.Index(pagesStr,"/Kids")
		kidx2 := 0
		for j:=kidx1; j<len(pagesBuf); j++ {
			if (pagesBuf[j] == '\r') || (pagesBuf[j] == '\n') {
				kidx2 = j
				break
			}
		}
		kidStr := pagesStr[kidx1:kidx2]
		fmt.Printf("kid str: %d %d %s\n",kidx1, kidx2, kidStr)
		dkidx1 := strings.Index(kidStr,"[")
		dkidx2 := strings.Index(kidStr,"]")
		fmt.Printf("kid data str: %d %d %s\n",dkidx1, dkidx2, kidStr[dkidx1:dkidx2+1])

/*
		pgCount, err := strconv.Atoi(pagesFlds[i])
		if err != nil {
			outstr = fmt.Sprintf("***error*** cannot convert Count %s in Pages to int!%v\n", pagesFlds[1], err)
			outfil.WriteString(outstr)
			return fmt.Errorf("error AnalysePdf:: cannot convert Count %s in Pages to int!%v", pagesFlds[1], err)
		}
//		pdf.pageCount = pgCount
*/

	} else {
		outfil.WriteString("***error*** pages object does not contain the attribute \"Kids\"!\n")
	}
// objects
	for i:=1;i<pdf.numObj; i++ {
		obj := (*pdf.objList)[i]
		strend := obj.end
		if (obj.end - obj.st) > 40 {strend = obj.st + 40}
		objStr := string(buf[obj.st:strend])
		fmt.Printf("obj %3d: %d %d %s\n", i, obj.st, strend, objStr)
		ntyp := -2
		switch {
		case strings.Contains(objStr,"/Catalog"):
			ntyp = 2
		case strings.Contains(objStr,"/Pages"):
			ntyp = 3
		case strings.Contains(objStr,"/Page"):
			ntyp = 4
		case strings.Contains(objStr,"/FontDescriptor"):
			ntyp = 6
		case strings.Contains(objStr,"/Font"):
			ntyp = 5
		case strings.Contains(objStr,"/Title"):
			fmt.Printf("title\n")
			ntyp = 1
		case strings.Contains(objStr,"/Filter"):
			ntyp = 7
		case strings.Contains(objStr,"/ca"):
			ntyp = 8
		default:
			ntyp = -1
		}
//		fmt.Printf("ntype: %d\n",ntyp)
		(*pdf.objList)[i].objTyp = ntyp
	}

	fmt.Printf("********Objects**********\n")
	for i:=1;i<pdf.numObj; i++ {
		obj := (*pdf.objList)[i]
		fmt.Printf("obj [%4d]: %5d %2d %-10s\n",i, obj.st, obj.objTyp, getObjTypStr(obj.objTyp))
	}
    (pdf.fil).Close()

// trying to decode
	fmt.Printf("\n**** obj 12 ****\n")
	obj := (*pdf.objList)[12]
	dbuf := buf[obj.st:obj.end]
	dbufStr := string(dbuf)
	didx := strings.Index(dbufStr, "/Length")
	didx2 := strings.Index(dbufStr, ">>")
//	fmt.Printf("l str: %s\n",dbufStr[didx +len("/Length "):didx2])
	obj12len, err:= strconv.Atoi(dbufStr[didx +len("/Length "):didx2])
	if err != nil {
		outstr = fmt.Sprintf("***error*** cannot convert obj12 length!%v\n", err)
		outfil.WriteString(outstr)
		return fmt.Errorf("error AnalysePdf:: cannot convert obj12 length!%v", err)
	}
	fmt.Printf("obj12 length: %d\n",obj12len)

	stidxst := strings.Index(dbufStr,"stream")
	stidxend := strings.Index(dbufStr,"endstream")
	streamStr := dbufStr[stidxst+len("stream")+1:stidxend-1]
	fmt.Printf("Stream: %d\n",len(streamStr))
	stbuf := []byte(streamStr)

	b := bytes.NewReader(stbuf)

	r, err := zlib.NewReader(b)
	if err != nil {
		panic(err)
	}
	nbuf := new(strings.Builder)
	io.Copy(nbuf, r)

	// check errors
//	fmt.Println(nbuf.String())
	r.Close()
	fmt.Printf("***obj 12:\n%s\n****\n", nbuf.String())

	fmt.Printf("\n**** obj 5 content ****\n")
	obj5 := (*pdf.objList)[5]
	dbuf = buf[obj5.st:obj5.end]
	dbufStr = string(dbuf)
	didx = strings.Index(dbufStr, "/Length")
	didx2 = strings.Index(dbufStr, ">>")
//	fmt.Printf("l str: %s\n",dbufStr[didx +len("/Length "):didx2])
	obj5len, err:= strconv.Atoi(dbufStr[didx +len("/Length "):didx2])
	if err != nil {
		outstr = fmt.Sprintf("***error*** cannot convert obj5 length!%v\n", err)
		outfil.WriteString(outstr)
		return fmt.Errorf("error AnalysePdf:: cannot convert obj5 length!%v", err)
	}
	fmt.Printf("obj5 length: %d\n",obj5len)

	stidxst = strings.Index(dbufStr,"stream")
	stidxend = strings.Index(dbufStr,"endstream")
	streamStr = dbufStr[stidxst+len("stream")+1:stidxend-1]
	fmt.Printf("Stream: %d\n",len(streamStr))
	stbuf = []byte(streamStr)

	b5 := bytes.NewReader(stbuf)

	r5, err := zlib.NewReader(b5)
	if err != nil {
		fmt.Printf("deflate error: %v\n", err)
		panic(err)
	}
	nbuf = new(strings.Builder)
	io.Copy(nbuf, r5)

	// check errors
//	fmt.Println(nbuf.String())
	r.Close()
	fmt.Printf("***obj 5:\n%s\n****\n", nbuf.String())


	fmt.Printf("\n**** obj 9 ****\n")
	obj9 := (*pdf.objList)[9]
	dbuf = buf[obj9.st:obj9.end]
	dbufStr = string(dbuf)
	didx = strings.Index(dbufStr, "/Length ")
	didx2 = strings.Index(dbufStr, ">>")
//	fmt.Printf("l str: %s\n",dbufStr[didx +len("/Length "):didx2])
	obj9len, err:= strconv.Atoi(dbufStr[didx +len("/Length "):didx2])
	if err != nil {
		outstr = fmt.Sprintf("***error*** cannot convert obj9 length!%v\n", err)
		outfil.WriteString(outstr)
		return fmt.Errorf("error AnalysePdf:: cannot convert obj9 length!%v", err)
	}
	fmt.Printf("obj9 length: %d\n",obj9len)

	stidxst = strings.Index(dbufStr,"stream")
	stidxend = strings.Index(dbufStr,"endstream")
	streamStr = dbufStr[stidxst+len("stream")+1:stidxend-1]
	fmt.Printf("Stream: %d\n",len(streamStr))
	stbuf = []byte(streamStr)

	b9 := bytes.NewReader(stbuf)

	r9, err := zlib.NewReader(b9)
	if err != nil {
		fmt.Printf("deflate error: %v\n", err)

	}
	nbuf = new(strings.Builder)
	io.Copy(nbuf, r9)

	// check errors
	r9.Close()
//	fmt.Printf("***obj 9:\n%s\n****\n", nbuf.String())



	return nil
}

func getObjTypStr(ityp int)(s string) {
	switch ityp {
		case 1:
			s = "Title"
		case 2:
			s = "Catalog"
		case 3:
			s = "Pages"
		case 4:
			s = "Page"
		case 5:
			s = "Font"
		case 6:
			s = "Font Desc"
		case 7:
			s = "Data"
		case 8:
			s= "ca"
		default:
			s= "unknown"
	}
	return s
}
