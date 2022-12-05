// library to createpdf.readLine
// analyse pdf documents
// author: prr
// created: 2/12/2022
//
// library pdf files in go
// author: prr
// date 2/12/2022
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
	util "prrpdf/utilLib"
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

const (
	Catalog = iota
	Pages
	Page
	Contents
	Font
	FontDescriptor
	Data
)

type InfoPdf struct {
	fil *os.File
	buf *[]byte
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
	objStart int
	pageIds []int
	pageList []pageObj
	objList *[]pdfObj
	rdObjList *[]pdfObj
	pages *pagesObj
//	doc pdfDoc
}

type pdfObj struct {
	objId int
	typ int
	typstr string
	dict bool
	array bool
	simple bool
	names *[]string
	parent int
	start int
	end int
	contSt int
	contEnd int
	streamSt int
	streamEnd int
}

type pagesObj struct {
	kids []int
	Font string
	mediaBox [4]int
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

func (pdf *InfoPdf) parseTrailer()(err error) {

	if pdf.startxref < 1 {return fmt.Errorf("no valid startxref")}
	if pdf.trailer < 1 {return fmt.Errorf("no valid trailer")}

	buf := *pdf.buf
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
		return fmt.Errorf("cannot find closing angular bracket for trailer!")
	}

	trailStart := 0
	for i:=pdf.trailer + 7; i< trailEnd; i++ {
		if buf[i] == '<' {
			if buf[i+1] == '<' {
				trailStart = i+2
				break
			}
		}
	}

	if trailStart == 0 {
		outstr := "// cannot find opening angular brackets for trailer!"
		return fmt.Errorf(outstr)
	}

//	fmt.Printf("trailer: %s\n", string(buf[trailStart:trailEnd]))

	objId, err := pdf.parseObjRef("Root",trailStart, 1)
	if err != nil {return fmt.Errorf("parse Root Obj error: %v!", err)}
	pdf.rootId = objId
//fmt.Printf("Root: %d\n", objId)

	objId, err = pdf.parseObjRef("Info",trailStart, 1)
	if err != nil {return fmt.Errorf("parse Info Obj error: %v!", err)}
	pdf.infoId = objId

	objId, err = pdf.parseObjRef("Size",trailStart, 2)
	if err != nil {return fmt.Errorf("parse Size Obj error: %v!", err)}
	pdf.numObj = objId


	return nil
}



func (pdf *InfoPdf) parseObjRef(key string, Start int, Type int)(objId int, err error) {

	//find key
	buf := *pdf.buf
	keyByt := []byte("/" + key)
	rootSt:=0

	linEnd := pdf.startxref

//fmt.Printf("trailer start: %d end: %d %s\n", Start, pdf.startxref, string(buf[Start:pdf.startxref]))

//fmt.Printf("str [%d:%d]: %s\n", Start, End, string(buf[Start:End]))
	for i:=Start; i< linEnd; i++ {
		ires := bytes.Index(buf[Start: linEnd], keyByt)
		if ires > -1 {
			rootSt = Start + ires
			break
		}
	}
	if rootSt == 0 {return -1, fmt.Errorf("cannot find keyword %s", key)}

	rootEnd :=0
	for i:=rootSt+4; i< linEnd; i++ {
		if buf[i] == '/' {
			rootEnd = i
			break
		}
		if buf[i] == '\n' || buf[i] == '\r' {
			rootEnd = i
			break
		}

	}
	if rootEnd == 0 {return -1, fmt.Errorf("cannot find obj after keyword %s!", key)}


	keyObjStr := string(buf[(rootSt + len(key) +1):rootEnd])

//fmt.Printf("%s: %s\n", key, keyObjStr)

	switch Type {
	case 1:
		val :=0
		_, err = fmt.Sscanf( keyObjStr, " %d %d R", &objId, &val)
		if err != nil {return -1, fmt.Errorf("cannot parse obj ref after keyword %s! %v", key, err)}
	case 2:
		_, err = fmt.Sscanf( keyObjStr, " %d", &objId)
		if err != nil {return -1, fmt.Errorf("cannot parse obj val after keyword %s! %v", key, err)}
	default:
		return -1, fmt.Errorf("invalid obj value type!")
	}
	return objId, nil
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
//fmt.Printf("i %d: %q\n",i ,instr[i])
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
func (pdf *InfoPdf) parseRootOld(instr string)(kvmap map[string]string, err error) {

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

func (pdf *InfoPdf) parseXref()(err error) {

	var pdfobj pdfObj
	var pdfObjList []pdfObj

	buf := *pdf.buf

	pdf.objList = &pdfObjList


	endStr := ""
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

	if pdf.xref <1 { return fmt.Errorf("not a valid xref supplied!")}
	if pdf.trailer <1 { return fmt.Errorf("not a valid trailer supplied!")}

	// check xref


	// replace with readLine ?
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
				outstr = fmt.Sprintf(" error parsing expected object heading [objid num] %s: %v", linStr, err)
				return fmt.Errorf(outstr)
			}
			totObj += objNum
			istate = 1
		case 1:
			_, err = fmt.Sscanf(linStr, "%d %d %s", &objSt, &val2, &endStr)
			if err != nil {
				outstr = fmt.Sprintf("   //error parsing object %d: %v", objCount, err)
				return fmt.Errorf(outstr)
			}
			if objCount > objNum {
				outstr = fmt.Sprintf("   //error too many obj ref %d: %v", objCount, err)
				return fmt.Errorf(outstr)
			}

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

	return nil
}

func (pdf *InfoPdf) parsePagesOld(instr string)(err error) {

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

	err = pdf.parseTrailer()
	trailerStr := "**** trailer ****\n"
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

//	pdfObjList, err := pdf.parseXref()

	outstr = xrefStr
//	outstr += objStr
	outstr += pEndStr
	txtFil.WriteString(outstr)

	// sort

	txtFil.WriteString("**************************\n")
	pdfObjList := *(pdf.objList)
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

	kvmap, err := pdf.parseRootOld(rootStr)
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
	err = pdf.parsePagesOld(pagesStr)
	if err != nil {
		outstr = fmt.Sprintf("// error parsing Pages: %v\n", err)
		txtFil.WriteString(outstr)
		return fmt.Errorf("error parsing Pages: %v", err)
	}

	outstr = fmt.Sprintf("Pages // Pages parsed successfully\n")
	outstr += fmt.Sprintf("page count: %d\n", pdf.pageCount)
	for i:=0; i< pdf.pageCount; i++ {
		outstr += fmt.Sprintf("page %d: id: %d\n",i+1 ,pdf.pageList[i])
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

func (pdf *InfoPdf) DecodePdf()(err error) {

	var outstr string

	buf := make([]byte,pdf.filSize)

	_, err = (pdf.fil).Read(buf)
	if err != nil {return fmt.Errorf("error Read: %v", err)}

	pdf.buf = &buf

	//read top two lines
	txtstr, nextPos, err := pdf.readLine(0)
	if err != nil {
		txtstr = fmt.Sprintf("// read top line: %v", err)
		return fmt.Errorf(txtstr)
	}
	outstr = txtstr + "\n"

	txtstr, nextPos, err = pdf.readLine(nextPos)
	if err != nil {
		txtstr = fmt.Sprintf("// read second top line: %v", err)
		return fmt.Errorf(txtstr)
	}

	// read last three lines

	bufLen := len(buf)

	ltStart := bufLen - 30
	sByt := []byte("startxref")

	ires := bytes.Index(buf[ltStart:], sByt)
	if ires < 0 {
		txtstr = "cannot find \"startxref\"!"
		return fmt.Errorf(txtstr)
	}

	ltStart += ires
	pdf.startxref = ltStart
//	fmt.Printf("ires %d\n%s\n", ires, string(buf[ltStart:]))

	txtstr, nextPos, err = pdf.readLine(ltStart)
	if err != nil {
		txtstr = fmt.Sprintf("// read third last line: %v", err)
		return fmt.Errorf(txtstr)
	}
	outstr += txtstr + "\n"

	txtstr, nextPos, err = pdf.readLine(nextPos)
	if err != nil {
		txtstr = fmt.Sprintf("// read second last line: %v", err)
		return fmt.Errorf(txtstr)
	}

	xref := 0
	_, err = fmt.Sscanf(txtstr, "%d", &xref)
	if err != nil {
		errconvStr := fmt.Sprintf("could not convert %s into xref: %v", txtstr, err)
		return fmt.Errorf(errconvStr)
	}
	pdf.xref = xref

//fmt.Printf("xref: %d\n", xref)

	// last line
	txtstr = string(buf[nextPos:])

	tStart := ltStart - 200
	tByt := []byte("trailer")

	tres := bytes.Index(buf[tStart:ltStart], tByt)
	if tres < 0 {
		txtstr = "cannot find \"trailer\"!"
		return fmt.Errorf(txtstr)
	}

	tStart += tres
	pdf.trailer = tStart

	txtstr, nextPos, err = pdf.readLine(tStart)
	if err != nil {
		txtstr = fmt.Sprintf("// read line with \"trailer\": %v", err)
		return fmt.Errorf(txtstr)
	}

	err = pdf.parseTrailer()
	if err != nil {
		txtstr = fmt.Sprintf("parseTrailor: %v", err)
		return fmt.Errorf(txtstr)
	}

	// trailer content

	xByt := []byte("xref")

	xres := bytes.Index(buf[pdf.xref:pdf.xref+7], xByt)
	if xres < 0 {
		txtstr = "cannot find \"xref\"!"
		return fmt.Errorf(txtstr)
	}

//fmt.Printf("xres: %d\n", xres)

	txtstr, nextPos, err = pdf.readLine(pdf.xref)
	if err != nil {
		txtstr = fmt.Sprintf("// read line with \"xref\": %v", err)
		return fmt.Errorf(txtstr)
	}

	// parse  Xref
	err = pdf.parseXref()
	if err != nil {return fmt.Errorf("parseXref: %v", err)}

	//list each object
	objList := *pdf.objList
	for i:=0; i< len(objList); i++ {
		_, err := pdf.decodeObjStr(i)
		if err != nil {
			txtstr = fmt.Sprintf("// getObjStr %d: %v", i, err)
			return fmt.Errorf(txtstr)
		}
	}

	// create pdf dom tree
	err = pdf.parseRoot()
	if err != nil {return fmt.Errorf("parseRoot: %v", err)}

	err = pdf.parsePages()
	if err != nil {return fmt.Errorf("parsePages: %v", err)}

	return nil
}

func (pdf *InfoPdf) DecodePdfToText(txtfil string)(err error) {
// method that decodes pdf file to text

	var outstr string

	txtFil, err := os.Create(txtfil)
	if err != nil {return fmt.Errorf("error creating textFile %s: %v\n", txtfil, err);}
	defer txtFil.Close()

	buf := make([]byte,pdf.filSize)

	_, err = (pdf.fil).Read(buf)
	if err != nil {return fmt.Errorf("error Read: %v", err)}

	pdf.buf = &buf

	// parsing sequence
	// Step 1: top 2 lines
	// Step 2: last 3 lines
	// Step 3: trailer section
	// Step 4: xref section
	// Step 5: objs
	//

	//read first line
	txtstr, nextPos, err := pdf.readLine(0)
	if err != nil {
		txtstr = fmt.Sprintf("// read top line: %v", err)
		txtFil.WriteString(txtstr + "\n")
		return fmt.Errorf(txtstr)
	}
	outstr = txtstr + "\n"

	// second line
	txtstr, nextPos, err = pdf.readLine(nextPos)
	if err != nil {
		txtstr = fmt.Sprintf("// read second top line: %v", err)
		txtFil.WriteString(outstr + txtstr + "\n")
		return fmt.Errorf(txtstr)
	}

	pdf.objStart = nextPos

	outstr += txtstr + "\n"

	txtFil.WriteString("********* first two lines ***********\n")
	txtFil.WriteString(outstr)
//fmt.Printf("**** first two lines ***\n%s\n",outstr)

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
	pdf.startxref = ltStart
//	fmt.Printf("ires %d\n%s\n", ires, string(buf[ltStart:]))

	txtstr, nextPos, err = pdf.readLine(ltStart)
	if err != nil {
		txtstr = fmt.Sprintf("// read third last line: %v", err)
		txtFil.WriteString(outstr + txtstr + "\n")
		return fmt.Errorf(txtstr)
	}
	outstr += txtstr + "\n"

	txtstr, nextPos, err = pdf.readLine(nextPos)
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

	// last line
	txtstr = string(buf[nextPos:])

	outstr += txtstr

	if buf[bufLen-1] != '\n' {outstr += "\n"}

	txtFil.WriteString(outstr)
//fmt.Printf("**** last three lines ***\n%s\n",outstr)


	//trailer
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
	pdf.trailer = tStart

	txtstr, nextPos, err = pdf.readLine(tStart)
	if err != nil {
		txtstr = fmt.Sprintf("// read line with \"trailer\": %v", err)
		txtFil.WriteString(outstr + txtstr + "\n")
		return fmt.Errorf(txtstr)
	}

	// txtstr should be "trailer"
	txtFil.WriteString(txtstr + "\n")

	// parses for root, info and size
	err = pdf.parseTrailer()
	if err != nil {
		txtstr = fmt.Sprintf("parse error \"trailer\": %v", err)
		txtFil.WriteString(txtstr + "\n")
		return fmt.Errorf(txtstr)
	}

	// trailer content
	txtstr = string(buf[nextPos:ltStart])
	txtFil.WriteString(txtstr)

	// xref section
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

	txtstr, nextPos, err = pdf.readLine(pdf.xref)
	if err != nil {
		txtstr = fmt.Sprintf("// read line with \"xref\": %v", err)
		txtFil.WriteString(outstr + txtstr + "\n")
		return fmt.Errorf(txtstr)
	}

	txtFil.WriteString(txtstr + "\n")

	// xref content
	txtstr = string(buf[nextPos:tStart])

	txtFil.WriteString(txtstr)

	// parse  Xref
	err = pdf.parseXref()
	if err != nil {return fmt.Errorf("parseXref: %v", err)}

	// pdf body consisting of a list of "obj"
	txtFil.WriteString("******** sequential obj list ***********\n")

	// get Objects directly
	rdObjList, err := pdf.readObjList()
	if err != nil {return fmt.Errorf("readObjList: %v", err)}
	pdf.rdObjList  = rdObjList
	txtFil.WriteString("Obj ObjId  start  end\n")
	for i:=0; i< len(*rdObjList); i++ {
		obj := (*rdObjList)[i]
		outstr = fmt.Sprintf("%3d [%3d]: %5d %5d\n", i, obj.objId, obj.start, obj.end)
		txtFil.WriteString(outstr)
	}

	objList := *pdf.objList

	//list each object
	for i:=0; i< len(objList); i++ {
		outstr = fmt.Sprintf("\n******** Obj %d ***********\n", i)
		txtFil.WriteString(outstr)

		objstr, err := pdf.decodeObjStr(i)
		if err != nil {
			txtstr = fmt.Sprintf("// getObjStr %d: %v", i, err)
			txtFil.WriteString(txtstr + "\n")
			return fmt.Errorf(txtstr)
		}
		txtFil.WriteString(objstr)
	}

	// collect objecs from xref
	txtFil.WriteString("******** xref obj list ***********\n")
	txtFil.WriteString("                             Content      Stream\n")
	txtFil.WriteString("Obj   Id type start  end   Start  End  Start  End  Length\n")
	for i:= 0; i< len(*pdf.objList); i++ {
		obj := (*pdf.objList)[i]
		outstr = fmt.Sprintf("%3d: %3d  %2d  %5d %5d %5d %5d %5d %5d %5d   %-15s\n",
		i, obj.objId, obj.typ, obj.start, obj.end, obj.contSt, obj.contEnd, obj.streamSt, obj.streamEnd, obj.streamEnd - obj.streamSt, obj.typstr)
		txtFil.WriteString(outstr)
	}
	txtFil.WriteString("\n")


	// create pdf dom tree
	err = pdf.parseRoot()
	if err != nil {return fmt.Errorf("parseRoot: %v", err)}
	txtFil.WriteString("parsed \"Root\" successfully!\n")

/*
	err = pdf.parsePages()
	if err != nil {return fmt.Errorf("parsePages: %v", err)}
	txtFil.WriteString("parsed \"Pages\" successfully!\n")
*/
	return nil
}

func (pdf *InfoPdf) readObjList()(rdobjList *[]pdfObj, err error) {

	var obj pdfObj
	var objList []pdfObj

	buf := *pdf.buf

	objSt:= pdf.objStart

	for i:=0;i<pdf.numObj; i++ {
		obj.start = objSt
		txtstr, nextPos, err := pdf.readLine(objSt)
		if err != nil {return nil, fmt.Errorf("readLine error for obj %d: %v", i, err)}
		if (nextPos - objSt) < 2 {
			objSt = nextPos + 1
			txtstr, nextPos, err = pdf.readLine(objSt)
		}
		if txtstr == "xref" {break}

		objId:= -1
		fmt.Sscanf(txtstr,"%d 0 obj", &objId)
		if objId == -1 {return nil, fmt.Errorf("obj %d no obj ref in %s found!", i, txtstr)}

		idx := bytes.Index(buf[objSt:pdf.xref],[]byte("endobj"))
		if idx == -1 {return nil, fmt.Errorf("no endobj in obj %d!",i+1)}
		objSt = objSt + idx + 7

		obj.objId = objId
		obj.end = objSt + idx + 6
		objList = append(objList, obj)
	}

//fmt.Printf("objs: %d\n",len(objList) )

	return &objList, nil
}


func (pdf *InfoPdf) parseObjLin(linSl []byte, istate int )(newstate int, err error) {

	var pdfobj pdfObj
	var objList []pdfObj

//fmt.Println("line: ",string(linSl))
	if (pdf.rdObjList) != nil {
		objList = *pdf.rdObjList
	}

	cObjIdx := len(objList) -1

	nxtLin := false
	for is:= 0; is<5; is++ {

		switch istate {
		case 0:
		// obj begin
			id:=0
			_, scerr := fmt.Sscanf(string(linSl), "%d 0 obj", &id)
			if scerr != nil {return istate, fmt.Errorf("obj start scan %s error: %v", string(linSl), scerr)}
			pdfobj.objId = id
			objList = append(objList, pdfobj)
			istate++
			nxtLin = true
		case 1:
		// obj type
			idx := bytes.Index(linSl, []byte("/Type"))
			if idx == -1 {
				objList[cObjIdx].typ = Data
			} else {
				tpos :=0
				for i:= 5; i< len(linSl);i++ {
					if linSl[i] == '/' {
						tpos = i
						break
					}
				}
				if tpos ==0 {return 0, fmt.Errorf("no type property found in %s", string(linSl))}
				tepos := len(linSl)-1
				for i:= tpos + 1; i< len(linSl);i++ {
					if linSl[i] == '/' || linSl[i] == ' ' {
						tepos = i
						break
					}
				}
				objTyp := string(linSl[tpos+1: tepos])
fmt.Printf("obj %d id %d type %s\n", cObjIdx, objList[cObjIdx].objId, objTyp)
			}
			istate++
			nxtLin = true
		case 2:
		// stream
			istate = 4
			nxtLin = false
		case 3:
		// endstream

		case 4:
		// end obj
			idx := bytes.Index(linSl, []byte("endobj"))
//return 0, fmt.Errorf("no endobj found in %s", string(linSl))}

			nxtLin = true
			if idx> -1 {istate = 0}
		default:
			return istate, fmt.Errorf("invalid istate: %d", istate)
		}
		if nxtLin {break}
	} // is
	pdf.rdObjList = &objList
	newstate = istate
	return newstate, nil
}

func (pdf *InfoPdf) parseRoot()(err error) {

	if pdf.rootId > pdf.numObj {return fmt.Errorf("invalid rootId!")}
	if pdf.rootId ==0 {return fmt.Errorf("rootId is 0!")}

	obj := (*pdf.objList)[pdf.rootId -1]

	objId, err := pdf.parseObjRef("Pages",obj.contSt, 1)
	if err != nil {return fmt.Errorf("Root obj: parse \"Pages\" error: %v!", err)}

	pdf.pagesId = objId

	return nil
}

func (pdf *InfoPdf) parsePages()(err error) {

	if pdf.pagesId > pdf.numObj {return fmt.Errorf("invalid pagesId!")}
	if pdf.pagesId ==0 {return fmt.Errorf("pagesId is 0!")}

	obj := (*pdf.objList)[pdf.pagesId -1]

	st := pdf.findKeyWord("Kids", obj)
	if st == -1 {
		outstr := "no keyword \"Kids\""
		return fmt.Errorf(outstr)
	}
	st = obj.contSt + st + 5


//fmt.Printf("kids St: %d %d %s\n", st, obj.contEnd, string((*pdf.buf)[st: st+25]))

	endPos := pdf.findVal(st, obj.contEnd)

//fmt.Printf("kids end %d: %d \n", st, st + endPos)

	valstr := string((*pdf.buf)[st:st+endPos])
fmt.Println("val str: ", valstr)
//	err = parseKids

	st = pdf.findKeyWord("MediaBox", obj)
	if st == -1 {
		outstr := "no keyword \"MediaBox\""
		return fmt.Errorf(outstr)
	}
fmt.Printf("Media Box St: %d\n", st)

	st = pdf.findKeyWord("Count", obj)
	if st == -1 {
		outstr := "no keyword \"Count\""
		return fmt.Errorf(outstr)
	}
fmt.Printf("Count St: %d\n", st)

	st = pdf.findKeyWord("Resources", obj)
	if st == -1 {
	}

fmt.Printf("Resources: %d\n", st)

	return nil
}

//aa
func (pdf *InfoPdf) findKeyWord(key string, obj pdfObj)(ipos int) {

	buf:= *pdf.buf
	objByt := buf[obj.contSt:obj.contEnd]
//fmt.Printf("find key: %s in %s\n", key, string(objByt))

	keyByt := []byte("/" + key)
	ipos = bytes.Index(objByt, keyByt)

	return ipos
}

func (pdf *InfoPdf) findVal(start, end int)(ipos int) {

	ipos = end
	buf:= *pdf.buf
	objByt := buf[start:end]

fmt.Printf("find val:%s \n", string(objByt))

	for i:=0; i< len(objByt); i++ {
		if objByt[i] == '/' {
			ipos = i
			break
		}
		if objByt[i] == '\n' || objByt[i] =='\r' {
			ipos = i
			break
		}
	}

fmt.Printf("find val end pos:%d \n", ipos)

	return ipos
}

func (pdf *InfoPdf) decodeObjStr(objId int)(outstr string, err error) {

	obj := (*pdf.objList)[objId]
	buf := *pdf.buf

	valst := -1
	valend := -1
	ipos :=-1
	xend := -1
	obj.streamSt = -1
	obj.streamEnd = -1

	// exception for info
	if objId+1 == pdf.infoId {obj.typstr = "Info"}
	(*pdf.objList)[objId] = obj

	// no vals for contSt and contEnd
	_, obj.contSt, err = pdf.readLine(obj.start)
	if err != nil {return "", fmt.Errorf("no eol for obj %d: %v", objId, err)}


	// find stream
	objByt := buf[obj.contSt:obj.end]
	xres := bytes.Index(objByt, []byte("stream"))
	if xres == -1 {
		obj.contEnd = obj.end -7
	} else {
		obj.streamSt = obj.contSt + xres
		obj.contEnd = obj.contSt + xres
	}

	objByt = buf[obj.contSt:obj.contEnd]

//fmt.Printf("\nobj: %d stream: %d [%d:%d]: %s\n", objId, obj.streamSt, obj.contSt, obj.contEnd, string(objByt))

	obj.simple = false

	// is dictionary
	xres = bytes.Index(objByt, []byte("<<"))

	if xres <0 {
		obj.dict = false
		goto endParse
	}

	obj.dict = true

	xend = bytes.LastIndex(objByt, []byte(">>"))
	if xend == -1 {
		outstr += "no closing brackets in dict of obj"
		return outstr + "\n", fmt.Errorf(outstr)
	}

	obj.contEnd = obj.contSt + xend +2

//fmt.Printf("obj %d dict after (<<>>) [%d:%d]: %s\n", objId, obj.contSt, obj.contEnd, string(buf[obj.contSt:obj.contEnd]))

	// check type
	ipos = bytes.Index(objByt, []byte("/Type"))
	if ipos == -1 {goto endParse}

	// has type
	for i:= ipos+5; i< len(objByt); i++ {
		if objByt[i] == '/' {
			valst = i
			break
		}
	}
	if valst == -1 {return "", fmt.Errorf("no name for /type in obj %d found!", objId)}

//fmt.Printf("\nafter /type: %s valst %d: %s\n", string(objByt[ipos:ipos+5]), valst, string(objByt[valst:20]))

	for i:= valst+1; i< len(objByt); i++ {
		switch objByt[i] {
		case '/', '\r', '\n':
			valend = i
		default:
		}
		if valend> -1 {break}
	}
	if valend == -1 {return "", fmt.Errorf("no eol for val of /type in obj %d found!", objId)}

	obj.typstr = string(objByt[(valst +1):valend])
//fmt.Printf("obj: %d valstr [%d:%d]: %s\n", objId, valst, valend, obj.typstr)


	(*pdf.objList)[objId] = obj

	endParse:

	//getstream
	outstr = string(objByt)
	if obj.streamSt < 0 {
		outstr += "no keyword stream"
		return outstr + "\n", nil
	}

	outstr += "has stream\n"

	xres = bytes.Index(buf[obj.streamSt:obj.end], []byte("endstream"))

	if xres == -1 {
		outstr += " cannot find \"endstream\"\n"
		return outstr + "\n", fmt.Errorf(outstr)
	}

	obj.streamEnd = obj.streamSt + xres

	(*pdf.objList)[objId] = obj

	return outstr, nil
}

func (pdf *InfoPdf) PrintPdf() {

	fmt.Printf("File Name: %s\n", pdf.filNam)
	fmt.Printf("File Size: %d\n", pdf.filSize)
	fmt.Println()

	fmt.Printf("Page Count: %3d\n", pdf.pageCount)
	fmt.Println()
	fmt.Printf("Info:       %5d\n", pdf.infoId)
	fmt.Printf("Root:       %5d\n", pdf.rootId)
	fmt.Printf("Pages:      %5d\n", pdf.pagesId)


	fmt.Printf("Xref:       %5d\n", pdf.xref)
	fmt.Printf("trailer:    %5d\n", pdf.trailer)
	fmt.Printf("startxref:  %5d\n", pdf.startxref)

	fmt.Println()
	fmt.Printf("*********** xref Obj List [%d] *********************\n", pdf.numObj)
	fmt.Printf("Objects: %5d Start: %7d\n", pdf.numObj, pdf.objStart)
	fmt.Println("********************************")

	fmt.Println("                             Content      Stream")
	fmt.Println("Obj   Id type start  end   Start  End  Start  End  Length")
	for i:= 0; i< len(*pdf.objList); i++ {
		obj := (*pdf.objList)[i]
		fmt.Printf("%3d: %3d  %2d  %5d %5d %5d %5d %5d %5d %5d   %-15s\n",
		i, obj.objId, obj.typ, obj.start, obj.end, obj.contSt, obj.contEnd, obj.streamSt, obj.streamEnd, obj.streamEnd - obj.streamSt, obj.typstr)
	}
	fmt.Println()
	fmt.Println("*********** sequential Obj List *********************")
	fmt.Println("Obj seq  Id start  type")
	for i:= 0; i< len(*pdf.rdObjList); i++ {
		obj := (*pdf.rdObjList)[i]
		fmt.Printf("%3d     %3d %5d\n", i, obj.objId, obj.start)
	}
	fmt.Println("************************************************")

	return
}

func (pdf *InfoPdf) readLine(stPos int)(outstr string, nextPos int, err error) {

	buf := *pdf.buf
//	bufLen := len(buf)

//fmt.Printf("\nreadLine (%d): %s\n", stPos, string(buf[stPos:stPos+20]))

//fmt.Println("********")
	endPos := -1

	maxPos := stPos + 3000
	if len(buf) < maxPos {maxPos = len(buf)}

	for i:=stPos; i < maxPos; i++ {
//		fmt.Printf("i: %d char: %q\n",i, buf[i])
		if buf[i] == '\n' || buf[i] == '\r'{
			endPos = i
			nextPos = i+1
			if buf[i+1] == '\n' {nextPos++}
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


func (pdf *InfoPdf) CreatePdfObj(pgNum int)(err error) {




//	pdf.objList = &pdfObjList
	return nil
}

func (pdf *InfoPdf) CreatePdf(pdfFilnam string)(err error) {

	var outstr string
	var pdfobj pdfObj
	var pdfObjList []pdfObj


	err = util.CheckFilnam(pdfFilnam, ".pdf")
	if err != nil {return fmt.Errorf("no pdf extension %s: %v\n", pdfFilnam, err);}

	pdfFil, err := os.Create(pdfFilnam)
	if err != nil {return fmt.Errorf("could not create pdf File %s: %v\n", pdfFilnam, err);}
	defer pdfFil.Close()

	// write top two lines
	tl := []byte("%pdf-1.4\n")
	sl := []byte{37,130, 131, 132, 10}

	bytSl := append(tl, sl...)
	pdfFil.Write(bytSl)

//	objSt := len(bytSl)


	// write objects

	var objSl []byte

	// info
	objStart, err := pdfFil.Seek(0, os.SEEK_END)
	if err != nil {return fmt.Errorf("seek %v", err)}
	objSt := int(objStart) + 1
	objBegin := int(objStart)
	objEnd := -1

	pdf.numObj++
	objStr := fmt.Sprintf("%d 0 obj\n", pdf.numObj)
	objLin := []byte(objStr)
	objSl = append(objSl, objLin...)

	objLin = []byte("<</Title (test)\n")
	objSl = append(objSl, objLin...)
	objLin = []byte("/Producer (azulPdf)>>\n")
	objSl = append(objSl, objLin...)

	objLin = []byte("endobj\n")
	objSl = append(objSl, objLin...)

	objEnd = objBegin + len(objSl)
//fmt.Printf("info: %d %d\n", objSt, objEnd)

	pdfobj.objId = pdf.numObj
	pdfobj.start = objSt
	pdfobj.end = objEnd
	pdfObjList = append(pdfObjList, pdfobj)
	pdf.infoId = pdf.numObj

	// root
	objSt = objEnd + 1
	objEnd = -1

	pdf.numObj++
	pdfobj.objId = pdf.numObj
	pdf.rootId = pdf.numObj
	objStr = fmt.Sprintf("%d 0 obj\n", pdf.numObj)
	objLin = []byte(objStr)

	objSl = append(objSl, objLin...)

	objLin = []byte("<</Type /Catalog\n")
	objSl = append(objSl, objLin...)

	objStr = fmt.Sprintf("/Pages %d 0 R>>\n", pdf.numObj +1)
	objLin = []byte(objStr)
//	objLin = []byte("/Pages 3 0 R>>\n")
	objSl = append(objSl, objLin...)

	objLin = []byte("endobj\n")
	objSl = append(objSl, objLin...)

//	pdfFil.Write(objSl)
	objEnd = objBegin + len(objSl)
//fmt.Printf("root: %d %d\n", objSt, objEnd)

	pdfobj.start = int(objSt)
	pdfobj.end = int(objEnd)
	pdfObjList = append(pdfObjList, pdfobj)

	// pages
	objSt = objEnd + 1
	objEnd = -1

	pdf.numObj++
	pdfobj.objId = pdf.numObj
	pdf.pagesId = pdf.numObj
	objStr = fmt.Sprintf("%d 0 obj\n", pdf.numObj)
	objLin = []byte(objStr)
//	objLin = []byte("3 0 obj\n")
	objSl = append(objSl, objLin...)

	objLin = []byte("<</Type /Pages\n")
	objSl = append(objSl, objLin...)

	// need to adjust for multiple pages

	objStr = fmt.Sprintf("/Kids [%d 0 R]\n", pdf.numObj +1)
	objLin = []byte(objStr)
//	objLin = []byte("/Kids [4 0 R]\n")
	objSl = append(objSl, objLin...)

	pgCount := 1

	objStr = fmt.Sprintf("/Count %d>>\n", pgCount)
	objLin = []byte(objStr)
//	objLin = []byte("/Count 1>>\n")
	objSl = append(objSl, objLin...)

	objLin = []byte("endobj\n")
	objSl = append(objSl, objLin...)

	objEnd = objBegin + len(objSl)

//fmt.Printf("objSt: %d %d\n", objSt, objEnd)

	pdfobj.start = int(objSt)
	pdfobj.end = int(objEnd)
	pdfObjList = append(pdfObjList, pdfobj)

	// page
	objSt = objEnd + 1
	objEnd = -1

	pdf.numObj++
	pdfobj.objId = pdf.numObj
	objStr = fmt.Sprintf("%d 0 obj\n", pdf.numObj)
	objLin = []byte(objStr)
//	objLin = []byte("4 0 obj\n")
	objSl = append(objSl, objLin...)

	objLin = []byte("<</Type /Page\n")
	objSl = append(objSl, objLin...)

	objStr = fmt.Sprintf("/Contents %d 0 R>>\n", pdf.numObj + 1)
	objLin = []byte(objStr)
//	objLin = []byte("/Contents 5 0 R>>\n")
	objSl = append(objSl, objLin...)

	objLin = []byte("endobj\n")
	objSl = append(objSl, objLin...)

	objEnd = objBegin + len(objSl)

//fmt.Printf("objSt: %d %d\n", objSt, objEnd)

	pdfobj.start = int(objSt)
	pdfobj.end = int(objEnd)
	pdfObjList = append(pdfObjList, pdfobj)

	// Content
	objSt = objEnd + 1
	objEnd = -1

	pdf.numObj++
	pdfobj.objId = pdf.numObj
	objStr = fmt.Sprintf("%d 0 obj\n", pdf.numObj)
	objLin = []byte(objStr)
//	objLin = []byte("5 0 obj\n")
	objSl = append(objSl, objLin...)

	objLin = []byte("<</Type /Contents\n")
	objSl = append(objSl, objLin...)

	streamLen := 99
	str := fmt.Sprintf("/Length %d\n", streamLen)
	objLin = []byte(str)
	objSl = append(objSl, objLin...)

	objLin = []byte("/Filter /FlateDecode>>\n")
	objSl = append(objSl, objLin...)


	objLin = []byte("stream\n")
	objSl = append(objSl, objLin...)
	// insert stream

	objLin = []byte("endstream\n")
	objSl = append(objSl, objLin...)


	objLin = []byte("endobj\n")
	objSl = append(objSl, objLin...)

	objEnd = objBegin + len(objSl)

//fmt.Printf("objSt: %d %d\n", objSt, objEnd)

	pdfobj.start = int(objSt)
	pdfobj.end = int(objEnd)
	pdfObjList = append(pdfObjList, pdfobj)


// Font
	objSt = objEnd + 1
	objEnd = -1

	pdf.numObj++
	pdfobj.objId = pdf.numObj
	objStr = fmt.Sprintf("%d 0 obj\n", pdf.numObj)
	objLin = []byte(objStr)
//	objLin = []byte("4 0 obj\n")
	objSl = append(objSl, objLin...)

	objLin = []byte("<</Type /Font\n")
	objSl = append(objSl, objLin...)
	objLin = []byte("/Subtype /TrueType\n")
	objSl = append(objSl, objLin...)
	objLin = []byte("/BaseFont /\"aaabbb + font\"\n")
	objSl = append(objSl, objLin...)
	objStr = fmt.Sprintf("/FontDescriptor %d 0 R>>\n", pdf.numObj + 1)
	objLin = []byte(objStr)


	objLin = []byte("endobj\n")
	objSl = append(objSl, objLin...)

	objEnd = objBegin + len(objSl)

//fmt.Printf("objSt: %d %d\n", objSt, objEnd)

	pdfobj.start = int(objSt)
	pdfobj.end = int(objEnd)
	pdfObjList = append(pdfObjList, pdfobj)

// Font Descriptor
	objSt = objEnd + 1
	objEnd = -1

	pdf.numObj++
	pdfobj.objId = pdf.numObj
	objStr = fmt.Sprintf("%d 0 obj\n", pdf.numObj)
	objLin = []byte(objStr)
	//	objLin = []byte("4 0 obj\n")
	objSl = append(objSl, objLin...)

	objLin = []byte("<</Type /FontDescriptor\n")
	objSl = append(objSl, objLin...)
	objLin = []byte("/FontName /\"aaabbb + font\"\n")
	objSl = append(objSl, objLin...)


	objStr = fmt.Sprintf("/FontFile %d 0 R>>\n", pdf.numObj + 1)
	objLin = []byte(objStr)


	objLin = []byte("endobj\n")
	objSl = append(objSl, objLin...)

	objEnd = objBegin + len(objSl)

//fmt.Printf("objSt: %d %d\n", objSt, objEnd)

	pdfobj.start = int(objSt)
	pdfobj.end = int(objEnd)
	pdfObjList = append(pdfObjList, pdfobj)

	// FontFile
	objSt = objEnd + 1
	objEnd = -1

	pdf.numObj++
	pdfobj.objId = pdf.numObj
	objStr = fmt.Sprintf("%d 0 obj\n", pdf.numObj)
	objLin = []byte(objStr)
	//	objLin = []byte("4 0 obj\n")
	objSl = append(objSl, objLin...)

	objLin = []byte("stream\n")
	objSl = append(objSl, objLin...)
	// insert stream

	objLin = []byte("endstream\n")
	objSl = append(objSl, objLin...)

	objLin = []byte("endobj\n")
	objSl = append(objSl, objLin...)

	objEnd = objBegin + len(objSl)

//fmt.Printf("objSt: %d %d\n", objSt, objEnd)

	pdfobj.start = int(objSt)
	pdfobj.end = int(objEnd)
	pdfObjList = append(pdfObjList, pdfobj)

	pdfFil.Write(objSl)

	// write xref
	xref, err := pdfFil.Seek(0, os.SEEK_END)
	if err != nil {return fmt.Errorf("xref pos %v", err)}
	pdf.xref = int(xref)

	pdfFil.WriteString("xref\n")
	outstr = fmt.Sprintf("0 %d\n", pdf.numObj + 1)
	pdfFil.WriteString(outstr)
	outstr = fmt.Sprintf("0000000000 65535 f \n")
	pdfFil.WriteString(outstr)

	objSl = []byte{}
	for i:=0; i<len(pdfObjList); i++ {
		objLin = []byte("0000000000 00000 n \n")
		str := fmt.Sprintf("%d",pdfObjList[i].start)
//fmt.Printf("obj %d: %s\n", i, str)
		pos := 10 -len(str)
		for j:=0; j<len(str); j++ {
			objLin[pos] = str[j]
			pos++
		}

		objSl = append(objSl, objLin...)
	}
	pdfFil.Write(objSl)

	// write trailer
	pdfFil.WriteString("trailer\n")
	outstr = fmt.Sprintf("<</Size %d\n", pdf.numObj)
	pdfFil.WriteString(outstr)
	outstr = fmt.Sprintf("/Root %d 0 R\n", pdf.rootId)
	pdfFil.WriteString(outstr)
	outstr = fmt.Sprintf("/Info %d 0 R>>\n", pdf.infoId)
	pdfFil.WriteString(outstr)

	// write last three lines
	pdfFil.WriteString("startxref\n")
	outstr = fmt.Sprintf("%d\n", pdf.xref)
	pdfFil.WriteString(outstr)
	outstr = "%%EOF"
	pdfFil.WriteString(outstr)
	return nil
}

