package main

import (
	"archive/tar"
	"compress/gzip"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
)

var (
	url = flag.String("url", "http://chaindata.tomotools.com:30304/20230128_CHAIN_DATA_block_60117599.tar.gz", "link download file")
)

func ExtractTarGz(gzipStream io.Reader) {
	uncompressedStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		log.Fatal("ExtractTarGz: NewReader failed")
	}

	tarReader := tar.NewReader(uncompressedStream)
	for true {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatalf("ExtractTarGz: Next() failed: %s", err.Error())
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.Mkdir(header.Name, 0755); err != nil {
				log.Fatalf("ExtractTarGz: Mkdir() failed: %s", err.Error())
			}
		case tar.TypeReg:
			outFile, err := os.Create(header.Name)
			if err != nil {
				log.Fatalf("ExtractTarGz: Create() failed: %s", err.Error())
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				log.Fatalf("ExtractTarGz: Copy() failed: %s", err.Error())
			}
			outFile.Close()

		default:
			log.Fatalf(
				"ExtractTarGz: uknown type: %s in %s",
				header.Typeflag,
				header.Name)
		}

	}
}
func main() {
	flag.Parse()
	resp, err := http.Get(*url)
	if err != nil {
		log.Fatalf("Error when download : %s ", err)
	}
	ExtractTarGz(resp.Body)
	defer resp.Body.Close()
}
