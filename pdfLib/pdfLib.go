// library to analyse pdf documents
// author: prr
// created: 18/2/2022
//
// 21/2/2022
// added zlib
//
// library pdf files in go
// author: prr
// date 29/2/2022
// copyright 2022 prr azul software
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
	sizeObj int
	numObj int
	xref int
	startxref int
	trailer int
	trail_root int
	trail_size int
	objList *[]pdfObj
	rootId int
	infoId int
	pagesId int
	pageCount int
	pglist []int
	pages pagesObj
	page []pageObj
}

type pdfObj struct {
	objId int
	objTyp int
	parent int
	start int
	end int
	contSt int
	contEnd int
}

type pagesObj struct {
	count int
	kids []int
}

type pageObj struct {
	pageNum int
	mediabox [4]int
	contentId int
	parentId int
//	fontObj int
}

type fontObj struct {

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


func (pdf *InfoPdf) parseTopTwoLines(buf []byte)(outstr string, err error) {

	//read top line
	end_fl :=0
	for i:=0; i<len(buf); i++ {
		if (buf[i] == '\n') {
			end_fl = i
			break
		}
	}
	if end_fl ==0 {
		outstr = "// no eol in first line!\n"
		return outstr, fmt.Errorf("no eol in first line!")
	}

	outstr = string(buf[:end_fl]) + "   // "

	if string(buf[:5]) != "%PDF-" {
		outstr += "no match to \"%%PDF-\" string in first line!\n"
		return outstr, fmt.Errorf("first line %s string is not \"%%PDF-\"!", string(buf[:5]))
	}

	verStr := string(buf[5:end_fl])
	version, err := strconv.ParseFloat(verStr, 32)
	if err != nil {
		outstr += fmt.Sprintf("version is not a valid float: %v!\n", err)
		return outstr, fmt.Errorf("error converting pdf version %s: %v", verStr, err)
	}

	if (version < 1.0) || (version > 2.0) {
		outstr += fmt.Sprintf("invalid pdf version %.2f\n", version)
		return outstr, fmt.Errorf("invalid pdf version %.2f", version)
	}

	outstr += "pdf version: " + verStr + " valid.\n"

	// second line
	end_sl :=0
	for i:=end_fl+1; i<len(buf); i++ {
		if (buf[i] == '\n') {
			end_sl = i
			break
		}
	}

	if end_sl ==0 {
		outstr += "// no eol in second line\n"
		return outstr, fmt.Errorf("no eol in second line!")
	}

	outstr += string(buf[end_fl+1:end_sl]) + "      // "

	start_sl := 0
	for i:=end_fl+ 1; i< end_sl; i++ {
		if buf[i] == '%' {
			start_sl = i
			break
		}
	}

	if start_sl == 0 {
		outstr += " No char % in second line\n"
		return outstr, fmt.Errorf("no % starting second line!")
	}

	if (end_sl - start_sl-1) != 4 {
		outstr += fmt.Sprintf(" No 4 chars after '%': %d!\n", end_sl - start_sl-1)
		return outstr, fmt.Errorf(" no 4 chars after '%' char!")
	}

	for i:=start_sl + 1; i< end_sl; i++ {
		if !(buf[i] > 120) {
			outstr += fmt.Sprintf(" char %q not valid!\n", buf[i])
			return outstr, fmt.Errorf("error char %q not valid!", buf[i])
		}
	}

	outstr += "second line valid!\n"

	return outstr, nil
}

func (pdf *InfoPdf) parseLast3Lines(inbuf *[]byte)(outstr string, err error) {

	buf := *inbuf
	llEnd := len(buf) -1

	for i:=llEnd; i>-1; i-- {
		if buf[i] != '\n' {
			llEnd = i
			break;
		}
	}

	if llEnd == len(buf) -1 {
		outstr = "// no beginning for last line\n"
		return outstr, fmt.Errorf("no beginning for last line!")
	}

	// now we have the real last line which should contain %%EOF
	start_ll :=0

	for i:=llEnd; i>llEnd - 20; i-- {
		if buf[i] == '\n' {
			start_ll = i+1
			break
		}
	}

	if start_ll ==0 {
		outstr = "// cannot find begin of last line!\n"
		return outstr, fmt.Errorf("cannot find begin of last line!")
	}

	llstr := string(buf[start_ll:llEnd+1])
	outstr = llstr + "      // "

	if llstr != "%%EOF" {
		outstr += fmt.Sprintf("last line not valid: %s!\n",llstr)
		return outstr, fmt.Errorf("last line %s is not \"%%EOF\"!", llstr)
	}

	outstr += "last line valid!\n"
//	txtFil.WriteString(outstr)

	//next we need to check second last line which should contain a int number
	sl_end:=start_ll-2
	if buf[sl_end] == '\n' {
		sl_end--
	}
	startxref_end :=0

	for i:=sl_end; i>0; i-- {
		if buf[i] == '\n' {
			startxref_end = i
			break
		}
	}
	if startxref_end ==0 {
		outstr += "// cannot find beginning of second to last line!\n"
		return outstr, fmt.Errorf("cannot find beginning of second to last line!")
	}

	slstr := string(buf[startxref_end+1:sl_end+1])
	xref, err := strconv.Atoi(slstr)
	if err != nil {
		outstr = slstr + fmt.Sprintf("     // cannot convert second last line %s to int: %v\n", slstr, err) + outstr
		return outstr, fmt.Errorf("second last line not an int: %s",slstr)
	}

	xrefstr := string(buf[xref: (xref+4)])
	if xrefstr != "xref" {
		outstr = slstr + fmt.Sprintf(" // xref does not point to xref string in file: getting %d %s! \n", len(xrefstr), xrefstr) +outstr
		return outstr, fmt.Errorf("xref pointer not pointing to xref: %s",slstr)
	}

	outstr = slstr + fmt.Sprintf("       // second last line valid pointer to xref %d\n", xref) + outstr

	//third last line
	// the third last line should have the word "startxref"
	tl_end := startxref_end -1
	// if the string ends with two chars /r + /n instead of one char
	if buf[tl_end] == '\n' {tl_end--}

	startxref_start :=0

	for i:=tl_end; i>0; i-- {
		if buf[i] == '\n' {
			startxref_start = i+1
			break
		}
	}
	if startxref_start ==0 {
		outstr = "// cannot find beginning of third to last line!\n" + outstr
		return outstr, fmt.Errorf("cannot find beginning to third last line!")
	}

	tlstr := string(buf[startxref_start:tl_end+1])
//	fmt.Printf("third last line: %s\n", tlstr)
	if tlstr != "startxref" {
		outstr = tlstr + " //  third line from end does not contain \"startxref\" keyword!" + tlstr + "\n" + outstr
		return outstr, fmt.Errorf("third line from end, s, does not contain \"startxref\" keyword! ", tlstr)
	}

	pdf.startxref = startxref_start
	outstr = tlstr + "  // valid third from end line\n" + outstr

	return outstr, nil
}

func (pdf *InfoPdf) parseTrailer(inbuf *[]byte)(outstr string, err error) {

	buf := *inbuf
	trailEnd := 0
	for i:=pdf.startxref; i> pdf.startxref-10; i-- {
		if buf[i] == '>' {
			if buf[i-1] == '>' {
				trailEnd = i
				break
			}
		}
	}

	if trailEnd == 0 {
		outstr = "// cannot find closing angular brackets for trailer!\n"
		return outstr, fmt.Errorf("cannot find closing angular bracket for trailer!")
	}

	trailStart := 0
	for i:=trailEnd; i> 0; i-- {
		if buf[i] == '<' {
			if buf[i-1] == '<' {
				trailStart = i
				break
			}
		}
	}

	if trailStart == 0 {
		outstr = "// cannot find opening angular brackets for trailer!\n"
		return outstr, fmt.Errorf("cannot find opening angular bracket for trailer!")
	}

	trailer_start :=0
	trailer_end :=0

	for i:=trailStart -1 ; i>trailStart - 21; i-- {
		if buf[i] == '\n' {
			trailer_end = i
			break
		}
	}

	if trailer_end == 0 {
		outstr = "// cannot find end of line with keyword \"trailer\"!\n"
		return outstr, fmt.Errorf("cannot find end of line with keyword \"trailer\"!")
	}

	// find beginning of line with key word trailer
	for i:=trailer_end -1 ; i>trailer_end - 21; i-- {
		if buf[i] == '\n' {
			trailer_start = i+1
			break
		}
	}
	if trailer_start ==0 {
		outstr = "// cannot find beginning of line with keyword \"trailer\"!\n"
		return outstr, fmt.Errorf("cannot find beginning of line with keyword \"trailer\"!")
	}

//	fmt.Printf("trailer start: %d end: %d\n", trailer_start, trailer_end)
	trailerStr := string(buf[trailer_start:trailer_end])
//	fmt.Printf("trailer line: %s\n", trailerStr)
	if trailerStr != "trailer" {
		outstr = fmt.Sprintf("// no keyword \"trailer\" found in %s!\n", trailerStr)
		return outstr, fmt.Errorf("no keyword \"trailer\" found in %s!", trailerStr)
	}

	pdf.trailer = trailer_start
//	trailStart--
//	trailEnd++
//	trailContentStr := string(buf[trailStart:trailEnd]) + "\n"
	outstr = trailerStr + "     // correct trailer header \n"

	// parseTrailContentStr
	trailCont := string(buf[trailStart+2:trailEnd-2]) + "\n"
//fmt.Printf("trailer Con:\n%s\n",trailCont)
	tConCount:=0
	linStr := ""
	ist := 0
	for i:=ist; i< len(trailCont); i++ {
		if trailCont[i] == '\n' {
			linStr = string(trailCont[ist:i])
			trailStr, err := pdf.parseTrailCont(linStr)
			if err != nil {
				outstr += trailStr
				return outstr, fmt.Errorf("could not parse trailer Content: %v", err)
			}
			ist = i+1
//fmt.Printf("trail line %d: %s\n", tConCount, linStr)
			tConCount++
			outstr += trailStr
		}
	}

	return outstr, nil
}

func (pdf *InfoPdf) parseTrailCont(linStr string)(outstr string, err error) {

	keyStr := string(linStr[1:5])
	switch keyStr {
	case "Size":
		size:= 0
		_, err = fmt.Sscanf(string(linStr[5:]),"%d",&size)
		if err != nil {
			outstr += fmt.Sprintf("Size: could not parse Size value: %v", err)
			return outstr, fmt.Errorf("could not parse Size value: %v", err)
		}
		pdf.sizeObj = size
		outstr += fmt.Sprintf(" Size: %d parsed correctly\n", size)

	case "Info":
		objId := 0
		val := 0
		_, err = fmt.Sscanf(string(linStr[5:]),"%d %d R",&objId, &val)
		if err != nil {
			outstr += fmt.Sprintf("Info: %s:: could not parse Info: %v", string(linStr[5:]), err)
			return outstr, fmt.Errorf("Info: could not parse value: %v", err)
		}
		pdf.infoId = objId
		outstr += fmt.Sprintf("Info: objId: %d  ref: %d R parsed successfully\n", objId, val)
	case "Root":
		objId := 0
		val := 0
		_, err = fmt.Sscanf(string(linStr[5:]),"%d %d R",&objId, &val)
		if err != nil {
			outstr += fmt.Sprintf("Root: %s:: could not parse Info: %v", string(linStr[5:]), err)
			return outstr, fmt.Errorf("Root: could not parse value: %v", err)
		}
		pdf.rootId = objId
		outstr += fmt.Sprintf("Root: objId: %d  ref: %d R parsed successfully\n", objId, val)

	default:
		outstr = fmt.Sprintf("%s is not a recognized keyword in Trailer\n", keyStr)
		return outstr, fmt.Errorf("invalid key word: %s", keyStr)
	}
	return outstr, nil
}

func (pdf *InfoPdf) getKVStr(instr string)(outstr string, err error) {

	stPos := -1
	endPos := 0

	for i:=0; i< len(instr); i++ {
		if instr[i] == '<' {
			if instr[i+1] == '<' {
				stPos = i+2
				break
			}
		}
	}

	if stPos == -1 {return "", fmt.Errorf("no open double bracket!")}

	for i:=len(instr)-1; i> stPos; i-- {
		if instr[i] == '>' {
			if instr[i-1] == '>' {
				endPos = i-2
				break
			}
		}
	}

	if endPos == 0 {return "", fmt.Errorf("no closing brackets!")}

	outstr = instr[stPos: endPos] + "\n"
	return outstr, nil
}

func (pdf *InfoPdf) getKvMap(instr string)(kvMap map[string]string , err error) {

fmt.Printf("******* getKvMap\n")
fmt.Println(instr)
fmt.Printf("******* getKvMap\n")
	ist := 0
	icount := 0
	linStr := ""
	key := ""
	val := ""
	kvMap = make(map[string]string)
	for i:=0; i< len(instr); i++ {
		if instr[i] == '\n' {
			linStr = instr[ist:i]
			ist = i+1
			icount++
fmt.Printf("linStr %d: %s\n", icount, linStr)
			_, err = fmt.Sscanf(linStr, "/%s %s", &key, &val)
			if err != nil {return kvMap, fmt.Errorf("parse error in line %d %s %v", icount, linStr, err)}
			// 2 : first letter is / second is ws
			val = linStr[(len(key)+2):]
fmt.Printf("key: %s val: %s\n", key, val)
			kvMap[key] = val
		}
	}

	if ist == 0 {return kvMap, fmt.Errorf("no eol found!")}

	return kvMap, nil
}

func (pdf *InfoPdf) getStream(instr string)(outstr string, err error) {

fmt.Printf("******* getstream instr\n")
fmt.Println(instr)
fmt.Printf("******* end ***\n")

	stPos := -1
	endPos := 0

//after stream there is a linefeed
	for i:=0; i< len(instr); i++ {
		if instr[i] == 's' {
			if string(instr[i:i+6]) == "stream" {
				stPos = i +7
				break
			}
		}
	}

	if stPos == -1 {return "", fmt.Errorf("no \"stream\" keyword found!")}

//	if instr[stPos +1] == '\n' {stPos = stPos +1}

	for i:=stPos-1; i< stPos+2; i++ {
		fmt.Printf("i %d: %q\n",i ,instr[i])
	}

	for i:=len(instr)-9; i> stPos; i-- {
		if instr[i] == 'e' {
			if string(instr[i:i+9]) == "endstream" {
				endPos = i -1
				break
			}
		}
	}

	if endPos == 0 {return "", fmt.Errorf("no \"endstream\" keyword found!")}

	outstr = instr[stPos: endPos]

fmt.Printf("stream len %d\n", len(outstr))
	return outstr, nil
}

// parseRoot parses ROOT object and returns a map of the object properties
func (pdf *InfoPdf) parseRoot(instr string)(kvmap map[string]string, err error) {

	kvmap, err = pdf.getKvMap(instr)
	if err != nil {return kvmap, fmt.Errorf("parseRoot: cannot get kv pairs: %v", err)}

	str, ok := kvmap["Type"]
	if !ok {return kvmap, fmt.Errorf("parseRoot: no Type property in Root object!")}
	if str != "/Catalog" {return kvmap, fmt.Errorf("parseRoot: type is not \"Catalog\": %s", str)}

	str, ok = kvmap["Pages"]
	if !ok {return kvmap, fmt.Errorf("parseRoot: no Pages property in Root object!")}

	if len(str ) > 10 {return kvmap, fmt.Errorf("parseRoot: value of Pages object %s is too long: %d!", string(str[1:10]) + "...", len(str))}

	pagesId :=0
	val := 0
	endStr :=""
	_, err = fmt.Sscanf(str,"%d %d %s", &pagesId, &val, &endStr)
	if err != nil {return kvmap, fmt.Errorf("parseRoot: cannot parse value %s of \"Pages\": %v", str, err)}

	if (pagesId < 1) || (pagesId> pdf.numObj) {return kvmap, fmt.Errorf("parseRoot: Pages object id outside range: %d", pagesId)}
	pdf.pagesId = pagesId

	return kvmap, nil
}

func (pdf *InfoPdf) parsePages(instr string)(pages *pagesObj, err error) {

//fmt.Printf("\n*****\nparsePages:\n%s\n***\n", instr)

	kvm, err := pdf.getKvMap(instr)
	if err != nil {return nil, fmt.Errorf("getVkMap error %v", err)}

	for key, val := range kvm {
		fmt.Printf("key: %s value: %s\n", key, val)
	}

	//Type
	val, ok := kvm["Type"]
	if !ok {return nil, fmt.Errorf("Pages: found no Type prop")}
	if val != "/Pages" {return nil, fmt.Errorf("Pages: Type prop is not Pages")}

	pages = new(pagesObj)
	// Count
	val, ok = kvm["Count"]
	if !ok {return nil, fmt.Errorf("Pages: found no Count prop")}
	count := 0
	_, err = fmt.Sscanf(val, "%d", &count)
	if err != nil {return nil, fmt.Errorf("Pages: cannot convert Count value!")}
	pdf.pageCount = count

	//kids
	val, ok = kvm["Kids"]
	if !ok {return nil, fmt.Errorf("Pages: found no Kids prop!")}

	pages.kids = make([]int, count)
	istate :=0
	stPos :=-1
	endPos :=-1
	pgCount :=0
	pgSt :=0
	pgId :=0
	pgRev :=0
	for i:=0; i< len(val); i++ {
		switch istate {
		case 0:
			if val[i] == '[' {
				stPos = i
				istate =1
				pgSt = i+1
			}
		case 1:
			if val[i] == 'R' {
				pgStr := string(val[pgSt:(i+1)])

				_, errPg := fmt.Sscanf(pgStr,"%d %d R",&pgId, &pgRev)
				if errPg != nil {return nil, fmt.Errorf("Pages: page %d str %s cannot be parsed: %v", pgCount, pgStr, errPg)}

				pages.kids[pgCount] = pgId
				pgSt = i+1
				pgCount++
				if (pgCount == count) {istate = 2}
			}

		case 2:
			if val[i] == ']' {
				endPos = i
				istate =3
			}
		case 3:
			break
		default:
		}
	} //i

	if stPos == -1 {return nil, fmt.Errorf("Pages: Kids val has no open bracket '['!")}
	if endPos == -1 {return nil, fmt.Errorf("Pages: Kids val has no closing bracket ']'!")}

	return pages, nil
}

func (pdf *InfoPdf) parsePage(instr string, pgNum int)(page *pageObj, err error) {

//fmt.Println("***** instr parsePage")
//fmt.Println(instr)
//fmt.Println("***** end instr")

	objStr, err := pdf.getKVStr(instr)
	if err != nil {return nil, fmt.Errorf("getVkStr error: %v", err)}

//fmt.Println("***** objStr parsePage")
//fmt.Println(objStr)
//fmt.Println("***** end objstr")

	kvm, err := pdf.getKvMap(objStr)
	if err != nil {return nil, fmt.Errorf("getVkMap error: %v", err)}

	for key, val := range kvm {
		fmt.Printf("key: %s value: %s\n", key, val)
	}

	page = new(pageObj)

	page.pageNum = pgNum

	objId := 0
	rev :=0

	//Type
	val, ok := kvm["Type"]
	if !ok {return nil, fmt.Errorf("Page: found no Type prop")}
	if val != "/Page" {return nil, fmt.Errorf("Page: Type prop is not Page")}

	//MediaBox
	val, ok = kvm["MediaBox"]
	if !ok {return nil, fmt.Errorf("Page: found no MediaBox prop")}

	//Contents
	val, ok = kvm["Contents"]
	if !ok {return nil, fmt.Errorf("Page: found no Contents prop")}
	_, errScan := fmt.Sscanf(val,"%d %d R", &objId, &rev)
	if errScan != nil {return nil, fmt.Errorf("Page: error parsing Contents: %v", errScan)}
	page.contentId = objId

	//Parent
	val, ok = kvm["Parent"]
	if !ok {return nil, fmt.Errorf("Page: found no Parent prop")}

	//Resources
	val, ok = kvm["Resources"]
	if !ok {return nil, fmt.Errorf("Pages: found no Resources prop")}


	return page, nil
}

func (pdf *InfoPdf) parseContent(instr string, pgNum int)(err error) {

	objStr, err := pdf.getKVStr(instr)
	if err != nil {return fmt.Errorf("getVkStr error: %v", err)}


fmt.Println("***** objStr parsePage")
fmt.Println(objStr)
fmt.Println("***** end objstr")

	kvm, err := pdf.getKvMap(objStr)
	if err != nil {return fmt.Errorf("getVkMap error: %v", err)}

	for key, val := range kvm {
		fmt.Printf("key: %s value: %s\n", key, val)
	}

fmt.Println("*** end Content kv ")

	streamStr, err := pdf.getStream(instr)

fmt.Printf("stream length: %d\n", len(streamStr))

	stbuf := []byte(streamStr)

	bytStream := bytes.NewReader(stbuf)

	bytR, err := zlib.NewReader(bytStream)
	if err != nil {return fmt.Errorf("stream deflate error: %v", err)}
	nbuf := new(strings.Builder)
	_, err = io.Copy(nbuf, bytR)
	if err != nil {return fmt.Errorf("stream deflate copy error: %v", err)}
	bytR.Close()

fmt.Printf("stream:\n%s\n****\n", nbuf.String())
fmt.Println("***** end streamstr")

	return nil
}

func (pdf *InfoPdf) CheckPdf(textFile string)(err error) {

	var outstr string

	txtFil, err := os.Create(textFile)
	if err != nil {return fmt.Errorf("error creating textFile %s: %v\n", textFile, err);}
	defer txtFil.Close()

	buf := make([]byte,pdf.filSize)

	_, err = (pdf.fil).Read(buf)
	if err != nil {return fmt.Errorf("error Read: %v", err)}

	// 40 character should be more than enough
	outstr , err = pdf.parseTopTwoLines(buf[:40])
	txtFil.WriteString(outstr)
	if err != nil {return fmt.Errorf("parseTopTwoLines: %v",err)}


	// last line
	// first get rid of empty lines at the end

	outstr , err = pdf.parseLast3Lines(&buf)
	txtFil.WriteString(outstr)
	if err != nil {return fmt.Errorf("parseLast3Lines: %v",err)}
	pEndStr := outstr

	return nil

	// find trailer
	outstr , err = pdf.parseTrailer(&buf)
	txtFil.WriteString(outstr + pEndStr)
	if err != nil {return fmt.Errorf("parseLast3Lines: %v",err)}

	pEndStr = outstr + pEndStr

	// find beginning of xref section
	trailer_start := pdf.trailer

	xref :=0
	for i:=trailer_start -1 ; i>0; i-- {
		if (buf[i] == 'x') {
			if string(buf[i: i+4]) == "xref" {
				xref = i
				break
			}
		}
	}

	if xref ==0 {
		outstr = fmt.Sprintf("// cannot find xref beginning of line!\n") + outstr
		return fmt.Errorf("cannot find xref beginning of line!")
	}

	xrefStr := fmt.Sprintf("xref       // found keyword xref at %d\n", xref)

	var pdfobj pdfObj
	var pdfObjList []pdfObj

	endStr := ""
	objStr := ""
	linStr := ""
	ist := xref + 5
	istate := 0
	objId := 0
	objIdNum := 0
	objCount := 0
	objSt := 0
	val2 := 0
	xrefErr := true
	totObj :=0

	for i:= ist; i < trailer_start; i++ {
		if buf[i] == '\n' {
			linStr = string(buf[ist:i])
			ist = i+1
			xrefErr = false
		} else {
			continue
		}
		switch istate {
		case 0:
			_, err1 := fmt.Sscanf(linStr, "%d %d", &objId, &objIdNum)
			if err1 != nil {
				outstr = xrefStr
				outstr += objStr
				outstr += fmt.Sprintf("   //error parsing expected object heading [objid num] %s: %v\n", linStr, err)
				outstr += pEndStr
				txtFil.WriteString(outstr)
				return fmt.Errorf("error parsing object heading %s: %v!", linStr, err)
			}
			totObj += objIdNum
			objStr += linStr + fmt.Sprintf("       // obj id %3d: number: %5d\n", objId, objIdNum)
			istate = 1
		case 1:
			_, err = fmt.Sscanf(linStr, "%d %d %s", &objSt, &val2, &endStr)
			if err != nil {
				outstr = xrefStr
				outstr += objStr
				outstr += fmt.Sprintf("   //error parsing object %d: %v\n", objCount, err)
				outstr += pEndStr
				txtFil.WriteString(outstr)
				return fmt.Errorf("error parsing object %d: %v!", objCount, err)
			}
			if objCount > objIdNum {
				outstr = xrefStr
				outstr += objStr
				outstr += fmt.Sprintf("   //error too many obj ref %d: %v\n", objCount, err)
				outstr += pEndStr
				txtFil.WriteString(outstr)
				return fmt.Errorf("error parsing object %d: %v!", objCount, err)
			}
			objStr += linStr + fmt.Sprintf(" // obj id %3d: start: %5d valid: %s\n", objId + objCount, objSt, endStr)
//xx
			pdfobj.objId = objId + objCount
			pdfobj.start = objSt
			if endStr == "n" {pdfObjList = append(pdfObjList, pdfobj)}
			if objCount == objIdNum {istate = 0}
			objCount++
		default:
		}

	} // i

	if xrefErr {
		outstr = xrefStr
		outstr += fmt.Sprintf("   //error no valid string after xref!\n")
		outstr += pEndStr
		txtFil.WriteString(outstr)
		return fmt.Errorf("no valid string (no lf) after xref!")
	}

	pdf.numObj = totObj

	outstr = xrefStr
	outstr += objStr
	outstr += pEndStr
	txtFil.WriteString(outstr)

	// sort
	for i:=0; i< len(pdfObjList); i++ {
		objEnd := xref
		for j:= 0; j<len(pdfObjList); j++ {
			if (pdfObjList[j].start < objEnd) && (pdfObjList[j].start>pdfObjList[i].start) {
				objEnd = pdfObjList[j].start
			}
		}
		pdfObjList[i].end = objEnd
	}

	txtFil.WriteString("**************************\n")
	pdfObjStr := ""
	for j:=0; j<len(pdfObjList); j++ {
		pdfObjStr += fmt.Sprintf("Obj Id: %4d Start: %5d End: %5d\n", pdfObjList[j].objId, pdfObjList[j].start, pdfObjList[j].end)
	}
	txtFil.WriteString(pdfObjStr)


	// check for obj and endobj
	for i:=0; i< len(pdfObjList); i++ {
		stp := pdfObjList[i].start
		contSt :=0
		contEnd :=0
		linstr :=""
		for j:= stp; j< stp + 80; j++ {
			if buf[j] == '\n' {
				linstr = string(buf[stp: j])
				contSt = j+1
				break
			}
		}
		if len(linstr) == 0 {
			txtFil.WriteString(fmt.Sprintf("Obj %d has no lf\n",i))
			return fmt.Errorf("Obj %d has no lf",i)
		}
//fmt.Printf("obj %d start: %s\n", i, linstr)
		objId :=0
		val := 0
		typStr :=""
		_, err = fmt.Sscanf(linstr,"%d %d %s",&objId, &val, &typStr)
		if err != nil {
			txtFil.WriteString(fmt.Sprintf("Obj %d cannot parse: %v\n", err))
			return fmt.Errorf("Obj %d cannot parse: %v", err)
		}
		if typStr != "obj" {
			txtFil.WriteString(fmt.Sprintf("Obj %d has no \"obj\" string: %s\n", typStr))
			return fmt.Errorf("Obj %d has no \"obj\" string: %s", typStr)
		}
		if objId != pdfObjList[i].objId {
			txtFil.WriteString(fmt.Sprintf("no match of Obj %d and objId %d \n", objId, pdfObjList[i].objId))
			return fmt.Errorf("no match of Obj %d and objId %d \n", objId, pdfObjList[i].objId)
		}

		// endobj
		endp := pdfObjList[i].end
		linstr =""
		for j:= endp - 2 ; j> stp; j-- {
			if buf[j] == '\n' {
				linstr = string(buf[j+1: endp -1])
				contEnd = j
				break
			}
		}

//fmt.Printf("obj %d end: %d %d %s\n", i, contEnd, endp, linstr)

		if linstr != "endobj" {
			txtFil.WriteString(fmt.Sprintf("Obj %d has no \"endobj\" string: %s\n", i, linStr))
			return fmt.Errorf("Obj %d has no \"endobj\" string: %s", i, linStr)
		}
//		fmt.Printf("obj %d:\n%s\n", i, buf[contSt:contEnd])
		pdfObjList[i].contSt = contSt
		pdfObjList[i].contEnd = contEnd
		// check for angular brackes


	} //i obj

	pdfObjStr = "*******\n"
	for j:=0; j<len(pdfObjList); j++ {
		pdfObjStr += fmt.Sprintf("num: %3d Obj Id: %4d Start: %5d End: %5d Content St: %5d End: %5d\n", j, pdfObjList[j].objId, pdfObjList[j].start, pdfObjList[j].end, pdfObjList[j].contSt, pdfObjList[j].contEnd)
	}
	txtFil.WriteString(pdfObjStr)


	// info object
	txtFil.WriteString("************Info**************\n")
	id := pdf.infoId - 1
	infoStr := string(buf[(pdfObjList[id].contSt+2):(pdfObjList[id].contEnd -2)]) + "\n"
	txtFil.WriteString(infoStr)
fmt.Printf("info:\n%s", infoStr)

	txtFil.WriteString("************ Root **************\n")
	id = pdf.rootId - 1
	rootStr := string(buf[(pdfObjList[id].contSt+2):(pdfObjList[id].contEnd -2)]) + "\n"
	txtFil.WriteString(rootStr)
fmt.Printf("************ root **************\n%s", rootStr)

	kvmap, err := pdf.parseRoot(rootStr)
	if err != nil {
		outstr = fmt.Sprintf("// error parsing Root: %v\n", err)
		txtFil.WriteString(outstr)
		return fmt.Errorf("error parsing Root: %v", err)
	}
	outstr = "  // ROOT properties\n"
	for key, val := range kvmap  {
		outstr += fmt.Sprintf("   %s: %s\n", key, val)
	}

	outstr += fmt.Sprintf("//Obj Root with id %d parsed successfully\n", pdf.rootId)
	txtFil.WriteString(outstr)

fmt.Printf("Obj ROOT with Obj id %d parsed successfully\n", id)
	// need to parse Pages
	txtFil.WriteString("************ Pages **************\n")
	id = 5
	pagesStr := string(buf[(pdfObjList[id].contSt+2):(pdfObjList[id].contEnd -2)]) + "\n"
	txtFil.WriteString(pagesStr)
fmt.Printf("pages:\n%s", pagesStr)

	pages, err := pdf.parsePages(pagesStr)
	if err != nil {
		outstr = fmt.Sprintf("// error parsing Pages: %v\n", err)
		txtFil.WriteString(outstr)
		return fmt.Errorf("error parsing Pages: %v", err)
	}
	outstr = fmt.Sprintf("// Pages parsed successfully\n%v\n", pages)
	txtFil.WriteString(outstr)

	// need to name each Page
	// Page

	for pg:=1; pg<(pdf.pageCount +1); pg++ {

		outstr = fmt.Sprintf("************ Page %d **************\n", pg)
		txtFil.WriteString(outstr)

		pageStr := string(buf[(pdfObjList[pg].contSt):(pdfObjList[pg].contEnd)]) + "\n"
		txtFil.WriteString(pageStr)
fmt.Printf("page 1:\n%s", pageStr)

		pagObj, err := pdf.parsePage(pageStr, pg)
		if err != nil {
			outstr = fmt.Sprintf("// error parsing Page %d: %v\n",pg ,err)
			txtFil.WriteString(outstr)
			return fmt.Errorf("error parsing Page: %v", err)
		}
		outstr = fmt.Sprintf("// Page %d parsed successfully\n%v\n",pg ,pagObj)
		txtFil.WriteString(outstr)


		// need to parse each Page
		outstr = fmt.Sprintf("************ Content Page %d **************\n", pg)
		txtFil.WriteString(outstr)

//		id = 4
		id = pagObj.contentId -1
fmt.Printf("page %d: contentId: %d\n", pg, id)
		contentStr := string(buf[(pdfObjList[id].contSt):(pdfObjList[id].contEnd)]) + "\n"

		// seperate stream and kv pairs
		err = pdf.parseContent(contentStr, pg)
		if err != nil {
			outstr = fmt.Sprintf("// error parsing Content %d for Page %d: %v\n",id, pg ,err)
			txtFil.WriteString(outstr)
			return fmt.Errorf("error parsing Content %d for Page %d: %v", id, pg, err)
		}

	}

	return nil

}

func (pdf *InfoPdf) GetPdfObjList(buf *[]byte)(objList *[]pdfObj, err error) {

	fmt.Println("****************GetPdfObjList************")
	bufLen := len(*buf)
	fmt.Printf("GetPdfObjList: %d\n", bufLen)
	fmt.Println("pdf Objects: ", pdf.numObj)
	objList = pdf.objList
	for i:= 1; i< pdf.numObj; i++ {
		fmt.Printf("obj: %d start: %d end %d\n",i,(*objList)[i].start, (*objList)[i].end)
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

	trailEnd := 0
	for i:=startxref_start; i> 0; i-- {
		if buf[i] == '>' {
			if buf[i-1] == '>' {
				trailEnd = i
				break
			}
		}
	}

	if trailEnd == 0 {
		outstr = fmt.Sprintf("***error*** cannot find closing angular brackets for trailer!\n")
		outfil.WriteString(outstr)
		return fmt.Errorf("error AnalysePdf: cannot find closing angular brackets for trailer!")
	}
	fmt.Printf("found closing angular brackets!\n")

	trailStart := 0
	for i:=trailEnd; i> 0; i-- {
		if buf[i] == '<' {
			if buf[i-1] == '<' {
				trailStart = i
				break
			}
		}
	}
	if trailStart == 0 {
		outstr = fmt.Sprintf("***error*** cannot find opening angular brackets for trailer!\n")
		outfil.WriteString(outstr)
		return fmt.Errorf("error AnalysePdf: cannot find opening angular brackets for trailer!")
	}
	fmt.Printf("found opening angular brackets! %s %d\n", string(buf[trailStart-1: trailStart+1]), trailStart-1)

	trailer_start :=0

	trailer_end :=0
	for i:=trailStart-2; i>0; i-- {
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
	trailCont := string(buf[trailStart -1:trailEnd+1])
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
	pdf.rootId, err = strconv.Atoi(rootFlds[1])
	if err!=nil {
		outstr = fmt.Sprintf("***error*** cannot convert root string %s to obj num in xref! %v\n", rootFlds[1], err)
		outfil.WriteString(outstr)
		return fmt.Errorf("error AnalysePdf:: cannot convert root string %s to object number in xref! %v", rootFlds[1], err)
	}
	outstr = fmt.Sprintf("Root: %d!\n", pdf.rootId)
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
	docObjList := make([]pdfObj, xrefObjnum)
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
			docObjList[xrefL].start = objPtr
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
	fmt.Printf("root: %d\n", pdf.rootId)

	outfil.WriteString("***key Objects***\n")
	outstr = fmt.Sprintf("root object: %d start: %d\n", pdf.rootId, (*pdf.objList)[pdf.rootId-1].start)
	outfil.WriteString(outstr)
	rootStr = string(buf[(*pdf.objList)[pdf.rootId].start:(*pdf.objList)[pdf.rootId].end])
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
	pagesStr := string(buf[(*pdf.objList)[pdf.pageCount].start:(*pdf.objList)[pdf.pageCount].end])
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
		if (obj.end - obj.start) > 40 {strend = obj.start + 40}
		objStr := string(buf[obj.start:strend])
		fmt.Printf("obj %3d: %d %d %s\n", i, obj.start, strend, objStr)
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
		fmt.Printf("obj [%4d]: %5d %2d %-10s\n",i, obj.start, obj.objTyp, getObjTypStr(obj.objTyp))
	}
    (pdf.fil).Close()

// trying to decode
	fmt.Printf("\n**** obj 12 ****\n")
	obj := (*pdf.objList)[12]
	dbuf := buf[obj.start:obj.end]
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
	dbuf = buf[obj5.start:obj5.end]
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
	dbuf = buf[obj9.start:obj9.end]
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

func (pdf *InfoPdf) ParsePdf()(err error) {
//	var outstr string

	buf := make([]byte,pdf.filSize)

	_, err = (pdf.fil).Read(buf)
	if err != nil {
		return fmt.Errorf("error Read: %v", err)
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
		return fmt.Errorf("invalid first line!")
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
	trailEnd := 0
	for i:=startxref_start; i> 0; i-- {
		if buf[i] == '>' {
			if buf[i-1] == '>' {
				trailEnd = i
				break
			}
		}
	}

	if trailEnd == 0 {
		return fmt.Errorf("error ParsePdf: cannot find closing angular bracket for trailer!")
	}
	fmt.Printf("found closing angular brackets!\n")

	trailStart := 0
	for i:=trailEnd; i> 0; i-- {
		if buf[i] == '<' {
			if buf[i-1] == '<' {
				trailStart = i
				break
			}
		}
	}
	if trailStart == 0 {
		return fmt.Errorf("error ParsePdf: cannot find opening angular bracket for trailer!")
	}
	fmt.Printf("found opening angular brackets! %s %d\n", string(buf[trailStart-1: trailStart+1]), trailStart-1)

	trailer_start :=0

	trailer_end :=0
	for i:=trailStart-2; i>0; i-- {
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

func (pdf *InfoPdf) ListObjs(textFile string)(err error) {

	buf := make([]byte,pdf.filSize)

	_, err = (pdf.fil).Read(buf)
	if err != nil {return fmt.Errorf("error Read: %v", err)}

	ist :=0
	linStr := ""
	objId:=0
	key:=""
	valStr := ""
	val:=0
	istate:=0
	objCount :=0
	for i:=0; i< len(buf); i++ {
		if buf[i] == '\n' {
			linStr = string(buf[ist:i])
			switch istate {
			case 0:
				_, errScan := fmt.Sscanf(linStr,"%d %d obj", &objId, &val)
				if errScan == nil {
					objCount++
					fmt.Printf("object %3d id %4d %2d at %5d", objCount, objId, val, i -len(linStr))
					istate = 1
				}
				ist = i+1
			case 1:
				_, errScan := fmt.Sscanf(linStr, "<</%s %s", &key, &valStr)
				if errScan == nil {
					fmt.Printf("  // type: %s prop: %s\n",key, valStr)
				} else {
					fmt.Printf("  // could not parse type: %v\n", errScan)
				}
				istate = 0
			default:

			}
		} //if

	}
	return nil
}
