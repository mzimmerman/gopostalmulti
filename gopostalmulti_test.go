package gopostalmulti

import (
	"log"
	"runtime"
	"testing"
)

func TestParse(t *testing.T) {
	log.Printf("%v", Parse("1265 East Fort Union Blvd., suite 250, Cotton Wood Hights, Utah 84047"))
}

var addresses = []string{
	"1607 23rd Street NW, Washington, DC 20008",
	"664 Dickens Road, Lilburn, Georgia 30047",
	"9520 Faires Farm Road, Charlotte, NC 28213",
	"526 Superior Avenue, Suite 1255, Cleveland, OH 44114",
	"2312 Dorrington Dr., Dallas, TX 75228",
	"2026 Scott Street, Hollywood, FL 33020",
	"318 Canino Road, Houston, TX 77076",
	"st 38th Street, New York, NY 10016",
	"77 Massachusetts Avenue, Cambridge, MA 02139",
	"1907 Spruce Street, Philadelphia, PA 19103-5732",
	"Michigan Avenue, Suite 1170, Chicago, IL 60611",
	"7280 N. Caldwell Avenue, Niles, IL 60714",
	"777 Woodward Avenue, Suite 300, Detroit, MI 48226",
	"1250 East Moore Lake Drive, Suite 242, Minneapolis (Fridley), MN 55432",
	"4761 Industrial Pkwy, Indianapolis, IN 46226",
	"11766 Wilshire Blvd., Suite 200, Los Angeles, CA 90025",
	"26050 Kay Ave., Hayward, CA 94545",
	"1350 Red Rock Street, Las Vegas, NV 89146",
	"5306 Walnut Ave., Building A, Sacramento, CA 95841",
	"Monroe, Suite C145, Phoenix, AZ 85004",
	"1151 SW Vermont, Portland, OR 97219",
	"1265 East Fort Union Blvd., suite 250, Cotton Wood Hights, Utah 84047",
}

func benchmarkParse(numAddresses int, b *testing.B) {
	b.ResetTimer()
	addressIn := make(chan string, 1000)
	for x := 0; x < runtime.NumCPU(); x++ {
		go func() {
			for add := range addressIn {
				Parse(add)
			}
		}()
	}
	for i := 0; i < b.N; i++ {
		for x := 0; x < numAddresses; x++ {
			addressIn <- addresses[x%len(addresses)]
		}
	}
	close(addressIn)
}

func BenchmarkParse50(b *testing.B) {
	benchmarkParse(50, b)
}

func BenchmarkParse500(b *testing.B) {
	benchmarkParse(500, b)
}

func BenchmarkParse5000(b *testing.B) {
	benchmarkParse(5000, b)
}

func BenchmarkParse50000(b *testing.B) {
	benchmarkParse(50000, b)
}
