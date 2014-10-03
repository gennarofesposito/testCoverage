/*
Ryan's notes:
you basically just need access to the test vpn (use the  vpn-eu.elasticride.com wrks for me )
http://jenkins01-global01-test.i.hailocab.com:3000/builds/names
http://jenkins01-global01-test.i.hailocab.com:3000/builds
http://jenkins01-global01-test.i.hailocab.com:3000/builds/com.hailocab.service.charging
http://jenkins01-global01-test.i.hailocab.com:3000/builds/com.hailocab.service.charging/coverage


Pseudocode as am so rusty
1) Read the list of names.json (or perhaps a csv from the ownership data "Component Ownership Matrix - Services and Owners Aug 2014.csv")
2) For each itemX in that list
3)	Go to http://jenkins01-global01-test.i.hailocab.com:3000/builds/itemX/coverage (very slow but give stats for all builds ever). This URL is [] empty for many services
4)		Cache it for a week for next time as it's a slow call
5)		Parse JSon and Take the most recent and oldest coverages from that in Json where the branch is 'master'
6)		Also work out the timestamp diff between first and last
7)			Parse Json the most recent coverage 
8)				For each 'PackageName:' in that initial coverage take the 'Percentage:'
9)				Where a  PackageName: exists in the most recent coverage, but not the oldest one : assume 0 initial coverage (ignore the case where the oldest has a package not in the newest)
10)				Draw a sparkline for each package
11)				Store this coverage data for week somehow as we cannot run http://jenkins01-global01-test.i.hailocab.com:3000/builds/itemX/coverage 130 times every day!
12)				Aggregate somehow for teams (implies the need for the ownership CSV file as the basis of step1)
13)				Be able to report averages by team (implies some kind of DB for storage / aggregation)
14)Retire to the bar

15 ) do a pointless PR to Rorie



)


*/


package main

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"log"
	"os"
	"fmt"
	"strconv"
	"time"
)

const CacheDirectoryPath = "./cachedData/"
const CacheFileExtension = ".json"
const UrlForCoveragesStub = "http://jenkins01-global01-test.i.hailocab.com:3000/builds/"
const UrlForCoveragesPostfix = "/coverage"
const PathtoOwnershipCSV = "./Component Ownership Matrix - Services and Owners Aug 2014.csv"
const CacheMaxAgeInDays = 3
const HoursInDay = 24
const BranchNameToChoose = "master"
const TargetJsonFileDirectory = "/Users/genoesposito/playing/angular-phonecat/jira/data/week"
const TargetJsonFileName = "/codeCoverage.json"

//
// Note: see this helpful article about referencing an array in a struct
// http://stackoverflow.com/questions/15945030/change-values-while-iterating-in-golang
//
type HailoService struct {
	ShortName  string
	Owner string
	Name string
	Packages []*CodePackage
}

type HailoServices struct {
	ServicesList []HailoService
}

type CodePackage struct {
	ShortName  string
	StartCoveragePercentage float64
	EndCoveragePercentage float64
	StartTimestamp int
	EndTimestamp int
}

//
// Create ownership matrix from the given reader
//
func NewOwnershipMatrixFromReader(reader io.Reader) (*HailoServices, error) {
	csvReader := csv.NewReader(reader)
	csvReader.TrailingComma = true
	rows, err := csvReader.ReadAll()
	if err != nil {
		return nil, err
	}
	servicesList := make([]HailoService, len(rows))
	for i, row := range rows {
		servicesList[i] = HailoService{row[0], row[1], row[2],[]*CodePackage{}}
	}
	return &HailoServices{servicesList}, nil
}

//
// Reads ownership matrix from the CSV on the path given
//
func NewOwnershipMatrixFromPath(path string) (*HailoServices, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return NewOwnershipMatrixFromReader(file)
}

//
// Tidies up error
//
func check(err error) {
	if err != nil {
		panic(err.Error())
	}
}

//
// Do all the things
//
// Note:  you need VPN access on for this next step to work as it hits Jenkins
// run the VPN access on your machine before running
//
func main() {
	//
	// Get the list of services from the product matrix (which I export as a CSV from the shared google doc)
	//
	hailoServices, err := NewOwnershipMatrixFromPath(PathtoOwnershipCSV)
	check(err)
	teamMap := make(map[string]HailoServices)
	//ownerServiceCountMap := make(map[string]int)
	//ownerPackageCountMap := make(map[string]int)
	//ownerServicesWithoutPackagesCountMap := make(map[string]int)
	//ownerScoreMap := make(map[string]float64)
	for i, thisService := range hailoServices.ServicesList {
		//
		// For now  skip the first line as its the header row in the CSV, 
		// the name being empty implies not H2 service and we're only talking about H2 here
		//			
		if i >0  && thisService.Name != "" {
			getAndParseCoverageJson(&thisService)
			// for _,currentPackage := range thisService.Packages {
			// 	//fmt.Printf("\nreport for name: %s packagesName: %s and start percentage: %v and endPercentage: %v \n", thisService.Name, currentPackage.ShortName, currentPackage.StartCoveragePercentage, currentPackage.endCoveragePercentage)
			// 	score += currentPackage.EndCoveragePercentage
			// }
			// //
			// // if there's no packages for the project assume there's one wiht zero coverage to make the averages better 
			// //
			// var packageLen = len(thisService.Packages)
			// if packageLen > 0 {
			// 	ownerPackageCountMap[thisService.Owner] += len(thisService.Packages)	
			// } else {
			// 	fmt.Printf("\nPackage frigging project: %s for team: %s\n",thisService.Name,thisService.Owner)
			// 	ownerServicesWithoutPackagesCountMap[thisService.Owner] += 1
			// }
			
			// ownerScoreMap[thisService.Owner] += score
			// ownerServiceCountMap[thisService.Owner] += 1

			//
			// check the map has an entry for our team, if not create
			//
			_, ok := teamMap[thisService.Owner] 
			if !ok {
				listServices := make([]HailoService, 0)
				//
				// incredibly you cannot allocate to a struct inside a map : how sh*t is go?
				//
				tmp := teamMap[thisService.Owner]
				tmp.ServicesList = listServices
				teamMap[thisService.Owner] = tmp
			}
			// 
			// Add the data sorted into teams
			// incredibly you cannot allocate to a struct inside a map : how sh*t is go?
			//
			tmp := teamMap[thisService.Owner]
			tmp.ServicesList = append (tmp.ServicesList, thisService) 
			teamMap[thisService.Owner] = tmp
		}
	}
	//for key, value := range ownerPackageCountMap {
    //	fmt.Printf("\n%s Team:: ServiceCount: %d PackageCount: %d total%%points:%d services without packages: %d  average%%coverage: %d \n", key, ownerServiceCountMap[key], value, int(ownerScoreMap[key]), ownerServicesWithoutPackagesCountMap[key], int(ownerScoreMap[key]/float64(value + ownerServicesWithoutPackagesCountMap[key])) )
	//}
	for key, value := range teamMap {
    	fmt.Printf("\n%s Team:: ServiceCount: %d PackageCount: %d total%%points:%d services without packages: %d  average%%coverage: %d \n", key, len(value.ServicesList) )
	}	
	year, week := time.Now().ISOWeek()
    fmt.Printf("Year %d\n", year)        
    //fmt.Printf("Week %d\n", week)
	fp, err := os.Create(TargetJsonFileDirectory + strconv.Itoa(week) + TargetJsonFileName)
    check(err)
    defer fp.Close()

    encoder := json.NewEncoder(fp)
    if err = encoder.Encode(teamMap); err != nil {
        log.Fatal("Unable to encode Json file. Err: %v.", err)
    }
}