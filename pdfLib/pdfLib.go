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

const (
	letter =iota
	landscape
	A5
	A4
	A3
	A2
	A1
)

type InfoPdf struct {
	fil *os.File
	filSize int64
	filNam string
	sizeObj int
	objSize int
	numObj int
	pageCount int
	infoId int
	rootId int
	pagesId int
	xref int
	startxref int
	trailer int
	pageIds []int
	pages []pageObj
	objList *[]pdfObj
//	doc pdfDoc
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
	kids []int
	defFont string
}

type pageObj struct {
	pageNum int
	mediabox [4]int
	contentId int
	parentId int
	fonts map[string]int
	extgstate map[string]int
}

type fontObj struct {

}

type pdfDoc struct {
	pageType int

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

	endFl := end_fl
	if buf[end_fl -1] == '\r' {endFl = endFl-1}

	outstr = string(buf[:endFl]) + "   // "

	if string(buf[:5]) != "%PDF-" {
		outstr += "no match to \"%%PDF-\" string in first line!\n"
		return outstr, fmt.Errorf("first line %s string is not \"%%PDF-\"!", string(buf[:5]))
	}

	verStr := string(buf[5:endFl])
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

	endSl := end_sl
	if buf[end_sl -1] == '\r' {endSl = endSl-1}
	outstr += string(buf[end_fl+1:endSl]) + "      // "

	start_sl := 0
	for i:=end_fl+ 1; i< endSl; i++ {
		if buf[i] == '%' {
			start_sl = i
			break
		}
	}

	if start_sl == 0 {
		outstr += " No char % in second line\n"
		return outstr, fmt.Errorf("no % starting second line!")
	}

	if (endSl - start_sl-1) != 4 {
		outstr += fmt.Sprintf(" No 4 chars after percent: %d!\n", endSl - start_sl-1)
		return outstr, fmt.Errorf(" no 4 chars after percent char %d %d!", endSl, start_sl)
	}

	for i:=start_sl + 1; i< endSl; i++ {
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

	bufEnd := len(buf) -1
//fmt.Printf("bufEnd: %d\n", bufEnd)
	// now we have the real last line which should contain %%EOF
	llStart :=0

	for i:=bufEnd; i>0; i-- {
		if buf[i] == '\n' {
//fmt.Printf("ll %d: %s\n", i, string(buf[i+1:i+6]))
			if string(buf[i+1:i+6]) == `%%EOF` {
				llStart = i +1
				break;
			}
		}
	}

	llEnd := llStart + 5

	if llStart ==0 {
		outstr = "// cannot find begin of last line!\n"
		return outstr, fmt.Errorf("cannot find begin of last line!")
	}

	llstr := string(buf[llStart:llEnd])
	outstr += llstr + "      // last line valid!\n"

	//next we need to check second last line which should contain a int number
	sl_end:=llStart -2
	if buf[sl_end] == '\r' { sl_end --}

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
		return outstr, fmt.Errorf("second last line not an int: %s", slstr)
	}

	outstr = slstr + fmt.Sprintf("       // second last line valid pointer to xref %d\n", xref) + outstr

	//third last line
	// the third last line should have the word "startxref"
	tl_end := startxref_end -1

	// if the string ends with two chars /r + /n instead of one char
	if buf[tl_end -1] == '\r' {tl_end--}


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


	tlstr := string(buf[startxref_start:tl_end])

	if int(pdf.filSize) < xref {
		outstr = tlstr + fmt.Sprintf(" //  startref points to invalid file location: %d file size: %d\n", xref, pdf.filSize ) + outstr
		return outstr, fmt.Errorf("startref points to invalid file location ", tlstr)
	}

	xrefstr := string(buf[xref: (xref+4)])
	if xrefstr != "xref" {
		outstr = slstr + fmt.Sprintf(" // xref does not point to xref string in file: getting %d %s! \n", len(xrefstr), xrefstr) +outstr
		return outstr, fmt.Errorf("xref pointer not pointing to xref: %s",slstr)
	}

//	fmt.Printf("third last line: %s\n", tlstr)
	if tlstr != "startxref" {
		outstr = tlstr + " //  third line from end does not contain \"startxref\" keyword!" + tlstr + "\n" + outstr
		return outstr, fmt.Errorf("third line from end %s does not contain \"startxref\" keyword! ", tlstr)
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
				trailEnd = i-1
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
				trailStart = i+1
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

	if buf[trailer_end -1] == '\r' {trailer_end--}
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

//fmt.Printf("trailer start: %d end: %d\n", trailer_start, trailer_end)
	trailerStr := string(buf[trailer_start:trailer_end])
//fmt.Printf("trailer line: %s\n", trailerStr)
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

	trailCont := string(buf[trailStart:trailEnd+1]) + "\n"
	tConCount:=0
	linStr := ""
	ist := 0
	for i:=ist; i< len(trailCont); i++ {
		if trailCont[i] == '\n' {
			linEnd := i
			if trailCont[i-1] == '\r' { linEnd--}
			linStr = string(trailCont[ist:linEnd])
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
		outstr += fmt.Sprintf("  Size: %d parsed correctly\n", size)

	case "Info":
		objId := 0
		val := 0
		_, err = fmt.Sscanf(string(linStr[5:]),"%d %d R",&objId, &val)
		if err != nil {
			outstr += fmt.Sprintf("  Info: %s:: could not parse Info: %v", string(linStr[5:]), err)
			return outstr, fmt.Errorf("Info: could not parse value: %v", err)
		}
		pdf.infoId = objId
		outstr += fmt.Sprintf("  Info: objId: %d  ref: %d R parsed successfully\n", objId, val)
	case "Root":
		objId := 0
		val := 0
		_, err = fmt.Sscanf(string(linStr[5:]),"%d %d R",&objId, &val)
		if err != nil {
			outstr += fmt.Sprintf("  Root: %s:: could not parse Info: %v", string(linStr[5:]), err)
			return outstr, fmt.Errorf("Root: could not parse value: %v", err)
		}
		pdf.rootId = objId
		outstr += fmt.Sprintf("  Root: objId: %d  ref: %d R parsed successfully\n", objId, val)

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
				endPos = i-1
				break
			}
		}
	}

	if endPos == 0 {return "", fmt.Errorf("no closing brackets!")}

	outstr = instr[stPos: endPos] + "\n"
	return outstr, nil
}

func (pdf *InfoPdf) getKvMap(instr string)(kvMap map[string]string , err error) {

//fmt.Printf("******* start getKvMap\n")
//fmt.Println(instr)
//fmt.Printf("******* end getKvMap\n")
	ist := 0
	icount := 0
	linStr := ""
	key := ""
	val := ""
	valStr := ""
	kvMap = make(map[string]string)

	for i:=0; i< len(instr); i++ {
		if instr[i] != '\n' { continue}

		linStr = instr[ist:i]
		icount++
//fmt.Printf("linStr %d: %s\n", icount, linStr)
			_, err = fmt.Sscanf(linStr, "/%s %s", &key, &val)
			if err != nil {return kvMap, fmt.Errorf("parse error in line %d %s %v", icount, linStr, err)}
			// 2 : first letter is / second is ws
//fmt.Printf("key: %s val: %s %q\n", key, val, val[0])

			switch val[0] {
			case '/':
				valStr = linStr[(len(key)+2):]
				ist = i+1

//fmt.Printf("    /valStr: %s\n", valStr)

			case '<':
				remSt := ist + len(key) +1
				remStr := string(instr[remSt:])
//fmt.Printf("remStr: %s %d\n", remStr, remSt)
				tvalStr, errStr := pdf.getKVStr(remStr)
				if errStr != nil {return kvMap, fmt.Errorf("parse error in line %s %v", remStr, errStr)}
//fmt.Printf("   <valStr: %s\n", valStr)
				ist = remSt + len(tvalStr) + 5
				i = ist + 6
				tvalByt := []byte(tvalStr)
				for j:=0; j< len(tvalByt); j++ {
					if tvalByt[j] == '\n' {tvalByt[j] = ' '}
				}
				valStr = string(tvalByt)
			default:
				valStr = linStr[(len(key)+2):]
				ist = i+1
//fmt.Printf("    def valStr: %q %s\n", val[0], valStr)

			}


			kvMap[key] = valStr

	} // i



	if ist == 0 {return kvMap, fmt.Errorf("no eol found!")}

	return kvMap, nil
}

func (pdf *InfoPdf) getStream(instr string)(outstr string, err error) {

//fmt.Printf("******* getstream instr\n")
//fmt.Println(instr)
//fmt.Printf("******* end ***\n")

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
//		fmt.Printf("i %d: %q\n",i ,instr[i])
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

	if (pagesId < 1) || (pagesId> pdf.numObj) {return kvmap, fmt.Errorf("parseRoot: Pages object id outside range: %d", pagesId + 1)}
	pdf.pagesId = pagesId

	return kvmap, nil
}

func (pdf *InfoPdf) parseXref(inbuf *[]byte)(pdfObjList []pdfObj, objstr string, err error) {

	var pdfobj pdfObj

	buf := *inbuf
	endStr := ""
	objStr := ""
	linStr := ""
	ist := pdf.xref + 5
	istate := 0
	objId := 0
	objNum := 0
	objCount := 0
	objSt := 0
	val2 := 0
	totObj :=0

	outstr :=""

	for i:= ist; i < pdf.trailer; i++ {
		if buf[i] == '\n' {
			linEnd:= i
			if buf[i-1] == '\r' { linEnd--}
			linStr = string(buf[ist:linEnd])
			ist = i+1
		} else {
			continue
		}

		switch istate {
		case 0:
			_, err1 := fmt.Sscanf(linStr, "%d %d", &objId, &objNum)
			if err1 != nil {
				outstr += fmt.Sprintf("   //error parsing expected object heading [objid num] %s: %v\n", linStr, err)
				return pdfObjList, outstr, fmt.Errorf("error parsing object heading %s: %v!", linStr, err)
			}
			totObj += objNum
			objStr += linStr + fmt.Sprintf("       // obj id %3d: number: %5d\n", objId, objNum)
			istate = 1
		case 1:
			_, err = fmt.Sscanf(linStr, "%d %d %s", &objSt, &val2, &endStr)
			if err != nil {
				outstr += fmt.Sprintf("   //error parsing object %d: %v\n", objCount, err)
				return pdfObjList, outstr, fmt.Errorf("error parsing object %d: %v!", objCount, err)
			}
			if objCount > objNum {
				outstr += fmt.Sprintf("   //error too many obj ref %d: %v\n", objCount, err)
				return pdfObjList, outstr, fmt.Errorf("error parsing object %d: %v!", objCount, err)
			}
			objStr += linStr + fmt.Sprintf(" // obj id %3d: start: %5d valid: %s\n", objId + objCount, objSt, endStr)

			pdfobj.objId = objId + objCount
			pdfobj.start = objSt
			if endStr == "n" {pdfObjList = append(pdfObjList, pdfobj)}
			if objCount == objNum {istate = 0}
			objCount++
		default:
		}

	} // i


	pdf.numObj = totObj

	for i:=0; i< len(pdfObjList); i++ {
		objEnd := pdf.xref
		for j:= 0; j<len(pdfObjList); j++ {
			if (pdfObjList[j].start < objEnd) && (pdfObjList[j].start>pdfObjList[i].start) {
				objEnd = pdfObjList[j].start
			}
		}
		pdfObjList[i].end = objEnd
	}

	return pdfObjList, objStr, nil
}

func (pdf *InfoPdf) parsePages(instr string)(err error) {

//fmt.Printf("\n*****\nparsePages:\n%s\n***\n", instr)

	kvm, err := pdf.getKvMap(instr)
	if err != nil {return fmt.Errorf("getVkMap error %v", err)}

//	for key, val := range kvm {
//		fmt.Printf("key: %s value: %s\n", key, val)
//	}

	//Type
	val, ok := kvm["Type"]
	if !ok {return fmt.Errorf("Pages: found no Type prop")}
	if val != "/Pages" {return fmt.Errorf("Pages: Type prop is not Pages")}

	// Count
	val, ok = kvm["Count"]
	if !ok {return fmt.Errorf("Pages: found no Count prop")}
	count := 0
	_, err = fmt.Sscanf(val, "%d", &count)
	if err != nil {return fmt.Errorf("Pages: cannot convert Count value!")}

	pdf.pageCount = count
	pdf.pageIds = make([]int,pdf.pageCount)

	//kids
	val, ok = kvm["Kids"]
	if !ok {return fmt.Errorf("Pages: found no Kids prop!")}

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
				if errPg != nil {return fmt.Errorf("Pages: page %d str %s cannot be parsed: %v", pgCount, pgStr, errPg)}

				pdf.pageIds[pgCount] = pgId
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

	if stPos == -1 {return fmt.Errorf("Pages: Kids val has no open bracket '['!")}
	if endPos == -1 {return fmt.Errorf("Pages: Kids val has no closing bracket ']'!")}

	return nil
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
fmt.Println("**** end Page kvm ***")

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
	var m1,m2, m3, m4 int
	_, errScan := fmt.Sscanf(val,"[%d %d %d %d]", &m1, &m2, &m3, &m4)
	if errScan != nil {return nil, fmt.Errorf("Page: error parsing MediaBox: %v", errScan)}


	//Contents
	val, ok = kvm["Contents"]
	if !ok {return nil, fmt.Errorf("Page: found no Contents prop")}
	_, errScan = fmt.Sscanf(val,"%d %d R", &objId, &rev)
	if errScan != nil {return nil, fmt.Errorf("Page: error parsing Contents: %v", errScan)}
	page.contentId = objId

	//Parent
	val, ok = kvm["Parent"]
	if !ok {return nil, fmt.Errorf("Page: found no Parent prop")}

	//Resources
	val, ok = kvm["Resources"]
	if !ok {return nil, fmt.Errorf("Page: found no Resources prop")}


	return page, nil
}

func (pdf *InfoPdf) parseContent(instr string, pgNum int)(outstr string, err error) {

fmt.Println("**** Content ***")
	outstr = fmt.Sprintf("**** KV ****\n")

	objStr, err := pdf.getKVStr(instr)
	if err != nil {
		outstr += fmt.Sprintf("// getVkStr error: %v\n", err)
		return outstr, fmt.Errorf("getVkStr error: %v", err)
	}


//fmt.Println("***** objStr parsePage")
//fmt.Println(objStr)
//fmt.Println("***** end objstr")

	kvm, err := pdf.getKvMap(objStr)
	if err != nil {
		outstr += fmt.Sprintf("getVkMap error: %v", err)
		return outstr, fmt.Errorf("getVkMap error: %v", err)
	}


	for key, val := range kvm {
		outstr += fmt.Sprintf("key: %s value: %s\n", key, val)
		fmt.Printf("key: %s value: %s\n", key, val)
	}

fmt.Println("*** end Content kv ")

	streamStr, err := pdf.getStream(instr)
	if err != nil {
		outstr += streamStr + fmt.Sprintf("stream deflate error: %v\n", err)
		return outstr, fmt.Errorf("stream deflate error: %v", err)
	}
	if len(streamStr) == 0 {
		outstr += "no stream\n"
		return outstr, nil
	}

//	outstr += streamStr

fmt.Printf("stream length: %d\n", len(streamStr))
	outstr += fmt.Sprintf("**** stream [length: %d] ****\n", len(streamStr))

	stbuf := []byte(streamStr)

	bytStream := bytes.NewReader(stbuf)

	bytR, err := zlib.NewReader(bytStream)
	if err != nil {
		outstr += fmt.Sprintf("stream deflate error: %v\n", err)
		return outstr, fmt.Errorf("stream deflate error: %v", err)
	}
	nbuf := new(strings.Builder)
	_, err = io.Copy(nbuf, bytR)
	if err != nil {
		outstr += fmt.Sprintf("stream copy error: %v\n", err)
		return outstr, fmt.Errorf("stream copy error: %v", err)
	}

	bytR.Close()

	outstr += nbuf.String()

fmt.Printf("stream:\n%s\n****\n", nbuf.String())
fmt.Println("***** end streamstr")

	return outstr, nil
}



func (pdf *InfoPdf) convertEOL(filbuf []byte)(buf []byte) {

	//test eol
	for i:=0; i< 40; i++ {
		if filbuf[i] == '\n' {
			if filbuf[i-1] != '\r' {
fmt.Println(" EOL has no cr!")
				return filbuf
			}
		}
	}

	fmt.Println("EOL has cr! -- converting")
	buf = make([]byte,pdf.filSize)

	icount := 0
	for i:=0; i< len(filbuf); i++ {
		if filbuf[i] != '\r' {
			buf[icount] = filbuf[i]
			icount++
		}
	}


	fmt.Printf("Size old: %d new: %d\n", len(filbuf), icount)
//	fmt.Printf("last line: %s\n",
	fmt.Printf("last line: %q %q %q\n", buf[icount-3], buf[icount -2], buf[icount-1])
	return buf
}

func (pdf *InfoPdf) CheckPdf(textFile string)(err error) {

	var outstr string

	txtFil, err := os.Create(textFile)
	if err != nil {return fmt.Errorf("error creating textFile %s: %v\n", textFile, err);}
	defer txtFil.Close()

	buf := make([]byte,pdf.filSize)

	_, err = (pdf.fil).Read(buf)
	if err != nil {return fmt.Errorf("error Read: %v", err)}

//	buf := pdf.convertEOL(filbuf)

	// 40 character should be more than enough
	outstr , err = pdf.parseTopTwoLines(buf[:40])
	outstr = "**** top two lines ***\n" + outstr
	txtFil.WriteString(outstr)
	if err != nil {return fmt.Errorf("parseTopTwoLines: %v",err)}


	// last line
	// first get rid of empty lines at the end

	l3LinStr , err := pdf.parseLast3Lines(&buf)
	outstr = "**** last three lines ***\n"
	if err != nil {
		txtFil.WriteString(outstr + l3LinStr)
		return fmt.Errorf("parseLast3Lines: %v",err)
	}
	pEndStr := outstr + l3LinStr

	// find trailer

	trailerStr , err := pdf.parseTrailer(&buf)
	trailerStr = "**** trailer ****\n" + trailerStr
	if err != nil {
		txtFil.WriteString(trailerStr + pEndStr)
		return fmt.Errorf("parseLast3Lines: %v",err)
	}

	pEndStr = trailerStr + pEndStr

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

	pdf.xref = xref

	pdfObjList, objStr, err := pdf.parseXref(&buf)

	outstr = xrefStr
	outstr += objStr
	outstr += pEndStr
	txtFil.WriteString(outstr)

	// sort

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
			return fmt.Errorf("Obj %d has no eol",i)
		}

//fmt.Printf("obj %d start: %s\n", i, linstr)

		objId :=0
		val := 0
		_, err = fmt.Sscanf(linstr,"%d %d obj",&objId, &val)
		if err != nil {
			txtFil.WriteString(fmt.Sprintf("Obj %d cannot parse: %v\n", err))
			return fmt.Errorf("Obj %d cannot parse: %v", err)
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
			txtFil.WriteString(fmt.Sprintf("Obj %d has no \"endobj\" string: %s\n", i, linstr))
			return fmt.Errorf("Obj %d has no \"endobj\" string: %s", i, linstr)
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
	id := pdf.infoId - 1
	hdStr := fmt.Sprintf("************ Info [Obj: %d] **************\n", id+1)
	txtFil.WriteString(hdStr)
	infoStr := string(buf[(pdfObjList[id].contSt+2):(pdfObjList[id].contEnd -2)]) + "\n"
	txtFil.WriteString(infoStr)
fmt.Printf("info:\n%s", infoStr)

	//ROOT
	id = pdf.rootId - 1
	hdStr = fmt.Sprintf("************ Root [Obj: %d] **************\n", id+1)
	txtFil.WriteString(hdStr)
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

//fmt.Printf("Obj ROOT with Obj id %d parsed successfully\n", id)

	// Pages Obj

	id = pdf.pagesId -1
	hdStr = fmt.Sprintf("************ Pages [Obj: %d] **************\n", id+1)
	txtFil.WriteString(hdStr)

	pagesStr :=string(buf[(pdfObjList[id].contSt+2):(pdfObjList[id].contEnd -2)]) + "\n"
	txtFil.WriteString( "string: \n" + pagesStr + "***** end string\n")
fmt.Printf("************ pages **********\n")
fmt.Println(pagesStr)
	err = pdf.parsePages(pagesStr)
	if err != nil {
		outstr = fmt.Sprintf("// error parsing Pages: %v\n", err)
		txtFil.WriteString(outstr)
		return fmt.Errorf("error parsing Pages: %v", err)
	}

	outstr = fmt.Sprintf("Pages // Pages parsed successfully\n")
	outstr += fmt.Sprintf("page count: %d\n", pdf.pageCount)
	for i:=0; i< pdf.pageCount; i++ {
		outstr += fmt.Sprintf("page %d: id: %d\n",i+1 ,pdf.pages[i])
	}
	txtFil.WriteString(outstr)


	// Page
	for pg:=0; pg<(pdf.pageCount ); pg++ {

		id := pdf.pageIds[pg] -1
		hdstr := fmt.Sprintf("************ Page %d [Obj: %d] **************\n", pg+1, id+1)
		txtFil.WriteString(hdstr)

		pageStr := string(buf[(pdfObjList[id].contSt):(pdfObjList[id].contEnd)]) + "\n"
		txtFil.WriteString(pageStr)

fmt.Println(pageStr)

fmt.Printf("******** page %d Obj %d *************\n%s\n**************end pageStr ********\n",pg +1, id +1, pageStr)

		pagObj, err := pdf.parsePage(pageStr, pg)
		if err != nil {
			outstr = fmt.Sprintf("// error parsing Page %d: %v\n",pg ,err)
			txtFil.WriteString(outstr)
			return fmt.Errorf("error parsing Page: %v", err)
		}
		outstr = fmt.Sprintf("// Page %d parsed successfully\n%v\n",pg ,pagObj)
		txtFil.WriteString(outstr)


		// need to parse each Page
		id = pagObj.contentId -1
		hdstr = fmt.Sprintf("************ Content Page %d [Obj %d] **************\n", pg+1, id+1)
		txtFil.WriteString(hdstr)

//fmt.Printf("page %d: contentId: %d\n", pg, id)
		contentStr := string(buf[(pdfObjList[id].contSt):(pdfObjList[id].contEnd)]) + "\n"

		// seperate stream and kv pairs
		outstr, err = pdf.parseContent(contentStr, pg)
		if err != nil {
			outstr += fmt.Sprintf("// error parsing Content %d for Page %d: %v\n",id, pg ,err)
			txtFil.WriteString(outstr)
			return fmt.Errorf("error parsing Content %d for Page %d: %v", id, pg, err)
		}
		txtFil.WriteString(outstr)


	} // page

	return nil

}

func (pdf *InfoPdf) DecodePdf(txtfil string)(err error) {

	var outstr string

	txtFil, err := os.Create(txtfil)
	if err != nil {return fmt.Errorf("error creating textFile %s: %v\n", txtfil, err);}
	defer txtFil.Close()

	buf := make([]byte,pdf.filSize)

	_, err = (pdf.fil).Read(buf)
	if err != nil {return fmt.Errorf("error Read: %v", err)}


	//read top two lines
	txtstr, nextPos, err := pdf.readLine(&buf, 0)
	if err != nil {
		txtstr = fmt.Sprintf("// read top line: %v", err)
		txtFil.WriteString(txtstr + "\n")
		return fmt.Errorf(txtstr)
	}
	outstr = txtstr + "\n"

	txtstr, nextPos, err = pdf.readLine(&buf, nextPos)
	if err != nil {
		txtstr = fmt.Sprintf("// read second top line: %v", err)
		txtFil.WriteString(outstr + txtstr + "\n")
		return fmt.Errorf(txtstr)
	}
	outstr += txtstr + "\n"

	txtFil.WriteString(outstr)

	// read last three lines

	txtFil.WriteString("******** last three lines ***********\n")
	bufLen := len(buf)
	outstr = ""

	ltStart := bufLen - 30
	sByt := []byte("startxref")

	ires := bytes.Index(buf[ltStart:], sByt)
	if ires < 0 {
		txtstr = "cannot find \"startxref\"!"
		txtFil.WriteString(txtstr + "\n")
		return fmt.Errorf(txtstr)
	}

	ltStart += ires
//	fmt.Printf("ires %d\n%s\n", ires, string(buf[ltStart:]))

	txtstr, nextPos, err = pdf.readLine(&buf, ltStart)
	if err != nil {
		txtstr = fmt.Sprintf("// read third last line: %v", err)
		txtFil.WriteString(outstr + txtstr + "\n")
		return fmt.Errorf(txtstr)
	}
	outstr += txtstr + "\n"

	txtstr, nextPos, err = pdf.readLine(&buf, nextPos)
	if err != nil {
		txtstr = fmt.Sprintf("// read second last line: %v", err)
		txtFil.WriteString(outstr + txtstr + "\n")
		return fmt.Errorf(txtstr)
	}
	outstr += txtstr + "\n"

	xref := 0
	_, err = fmt.Sscanf(txtstr, "%d", &xref)
	if err != nil {
		errconvStr := fmt.Sprintf("could not convert %s into xref: %v", txtstr, err)
		txtFil.WriteString(outstr + "error: " + errconvStr + "\n")
		return fmt.Errorf(errconvStr)
	}
	pdf.xref = xref

//fmt.Printf("xref: %d\n", xref)

	// last line
	txtstr = string(buf[nextPos:])

	outstr += txtstr

	if buf[bufLen-1] != '\n' {outstr += "\n"}

	txtFil.WriteString(outstr)

	txtFil.WriteString("******** trailer ***********\n")

	outstr = ""
	tStart := ltStart - 200
	tByt := []byte("trailer")

	tres := bytes.Index(buf[tStart:ltStart], tByt)
	if tres < 0 {
		txtstr = "cannot find \"trailer\"!"
		txtFil.WriteString(txtstr + "\n")
		return fmt.Errorf(txtstr)
	}

	tStart += tres

	txtstr, nextPos, err = pdf.readLine(&buf, tStart)
	if err != nil {
		txtstr = fmt.Sprintf("// read line with \"trailer\": %v", err)
		txtFil.WriteString(outstr + txtstr + "\n")
		return fmt.Errorf(txtstr)
	}

	txtFil.WriteString(txtstr + "\n")

	// trailer content
	txtstr = string(buf[nextPos:ltStart])
	txtFil.WriteString(txtstr)

	txtFil.WriteString("******** xref ***********\n")
	outstr = ""
	xByt := []byte("xref")

	xres := bytes.Index(buf[pdf.xref:pdf.xref+7], xByt)
	if xres < 0 {
		txtstr = "cannot find \"xref\"!"
		txtFil.WriteString(txtstr + "\n")
		return fmt.Errorf(txtstr)
	}

//fmt.Printf("xres: %d\n", xres)

	txtstr, nextPos, err = pdf.readLine(&buf, pdf.xref)
	if err != nil {
		txtstr = fmt.Sprintf("// read line with \"xref\": %v", err)
		txtFil.WriteString(outstr + txtstr + "\n")
		return fmt.Errorf(txtstr)
	}

	txtFil.WriteString(txtstr + "\n")

	// xref content
	txtstr = string(buf[nextPos:tStart])

	txtFil.WriteString(txtstr)


	txtFil.WriteString("******** obj ***********\n")

	return nil
}

func (pdf *InfoPdf) readLine(inbuf *[]byte, stPos int)(outstr string, nextPos int, err error) {

	buf := *inbuf
//	bufLen := len(buf)

//fmt.Printf("\nreadLine (%d): %s\n", stPos, string(buf[stPos:stPos+20]))

//fmt.Println("********")
	endPos := -1

	maxPos := stPos + 20
	if len(buf) < maxPos {maxPos = len(buf)}

	for i:=stPos; i < maxPos; i++ {
//		fmt.Printf("i: %d char: %q\n",i, buf[i])
		if buf[i] == '\n' {
			endPos = i
			nextPos = i+1
			if buf[i-1] == '\r' {endPos = i-1}
			break
		}
	}
	if endPos == -1 {return "", -1, fmt.Errorf("no eol found!")}

	outstr = string(buf[stPos:endPos])

//fmt.Printf("out: %s next: %d\n", outstr, nextPos)
	return outstr, nextPos, nil
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


