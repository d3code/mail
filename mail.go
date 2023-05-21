package xmail

import (
    "bytes"
    "encoding/base64"
    "fmt"
    "io"
    "log"
    "mime"
    "mime/multipart"
    "mime/quotedprintable"
    "net/mail"
    "os"
    "strings"
)

// BuildFileName builds a file name for a MIME part, using information extracted from
// the part itself, as well as a radix and an index given as parameters.
func BuildFileName(part *multipart.Part, radix string, index int) (filename string) {

    // 1st try to get the true file name if there is one in Content-Disposition
    filename = part.FileName()
    if len(filename) > 0 {
        return
    }

    // If no default filename defined, try to build one of the following format :
    // "radix-index.ext" where extension is computed from the Content-Type of the part
    mediaType, _, err := mime.ParseMediaType(part.Header.Get("Content-Type"))
    if err == nil {
        mimeType, e := mime.ExtensionsByType(mediaType)
        if e == nil {
            return fmt.Sprintf("%s-%d%s", radix, index, mimeType[0])
        }
    }
    return
}

// WritePart decodes the data of MIME part and writes it to the file filename.
func WritePart(part *multipart.Part, filename string) {

    // Read the data for this MIME part
    partData, err := io.ReadAll(part)
    if err != nil {
        log.Println("Error reading MIME part data -", err)
        return
    }

    contentTransferEncoding := strings.ToUpper(part.Header.Get("Content-Transfer-Encoding"))

    switch {

    case strings.Compare(contentTransferEncoding, "BASE64") == 0:
        decodedContent, err := base64.StdEncoding.DecodeString(string(partData))
        if err != nil {
            log.Println("Error decoding base64 -", err)
        } else {
            err := os.WriteFile(filename, decodedContent, 0644)
            if err != nil {
                return
            }
        }

    case strings.Compare(contentTransferEncoding, "QUOTED-PRINTABLE") == 0:
        decodedContent, err := io.ReadAll(quotedprintable.NewReader(bytes.NewReader(partData)))
        if err != nil {
            log.Println("Error decoding quoted-printable -", err)
        } else {
            err := os.WriteFile(filename, decodedContent, 0644)
            if err != nil {
                return
            }
        }

    default:
        err := os.WriteFile(filename, partData, 0644)
        if err != nil {
            return
        }

    }

}

// ParsePart parses the MIME part from mime_data, each part being separated by
// boundary. If one of the part read is itself a multipart MIME part, the
// function calls itself to recursively parse all the parts. The parts read
// are decoded and written to separate files, named upon their Content-Description
// (or boundary if no Content-Description available) with the appropriate
// file extension. Index is incremented at each recursive level and is used in
// building the filename where the part is written, as to ensure all filenames
// are distinct.
func ParsePart(mimeData io.Reader, boundary string, index int) {

    // Instantiate a new io.Reader dedicated to MIME multipart parsing
    // using multipart.NewReader()
    reader := multipart.NewReader(mimeData, boundary)
    if reader == nil {
        return
    }

    fmt.Println(strings.Repeat("  ", 2*(index-1)), ">>>>>>>>>>>>> ", boundary)

    // Go through each of the MIME part of the message Body with NextPart(),
    // and read the content of the MIME part with io.ReadAll()
    for {

        newPart, err := reader.NextPart()
        if err == io.EOF {
            break
        }
        if err != nil {
            fmt.Println("Error going through the MIME parts -", err)
            break
        }

        for key, value := range newPart.Header {
            fmt.Printf("%s Key: (%+v) - %d Value: (%#v)\n", strings.Repeat("  ", 2*(index-1)), key, len(value), value)
        }
        fmt.Println(strings.Repeat("  ", 2*(index-1)), "------------")

        mediaType, params, err := mime.ParseMediaType(newPart.Header.Get("Content-Type"))
        if err == nil && strings.HasPrefix(mediaType, "multipart/") {
            ParsePart(newPart, params["boundary"], index+1)
        } else {
            filename := BuildFileName(newPart, boundary, 1)
            WritePart(newPart, "_test/out/"+filename)
        }

    }

    fmt.Println(strings.Repeat("  ", 2*(index-1)), "<<<<<<<<<<<<< ", boundary)

}

// Parse reads a MIME multipart email from stdio and explode its MIME parts into
// separated files, one for each part.
func Parse(message io.Reader) {

    log.SetFlags(log.LstdFlags | log.Lshortfile)

    //  Parse the message to separate the Header and the Body with mail.ReadMessage()
    m, err := mail.ReadMessage(message)
    if err != nil {
        log.Fatalln("Parse mail KO -", err)
    }

    // Display only the main headers of the message. The "From","To" and "Subject" headers
    // have to be decoded if they were encoded using RFC 2047 to allow non ASCII characters.
    // We use a mime.WordDecode for that.
    dec := new(mime.WordDecoder)
    from, _ := dec.DecodeHeader(m.Header.Get("From"))
    to, _ := dec.DecodeHeader(m.Header.Get("To"))
    subject, _ := dec.DecodeHeader(m.Header.Get("Subject"))
    fmt.Println("From:", from)
    fmt.Println("To:", to)
    fmt.Println("Date:", m.Header.Get("Date"))
    fmt.Println("Subject:", subject)
    fmt.Println("Content-Type:", m.Header.Get("Content-Type"))
    fmt.Println()

    mediaType, params, err := mime.ParseMediaType(m.Header.Get("Content-Type"))
    if err != nil {
        log.Fatal(err)
    }

    if !strings.HasPrefix(mediaType, "multipart/") {
        log.Fatalf("Not a multipart MIME message\n")
    }

    // Recursively parse the MIME parts of the Body, starting with the first
    // level where the MIME parts are separated with params["boundary"].
    ParsePart(m.Body, params["boundary"], 1)
}
