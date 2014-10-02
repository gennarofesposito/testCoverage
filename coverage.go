package main

import (
        "fmt"
        "net/http"
        "io/ioutil"
        "encoding/json"
        "os"
        "time"
        )

type CoverageInfo struct {
    Coverages []CoverageInfoItem
    Branch string
    Version string
    Timestamp int
}

type CoverageInfoItem struct {
    PackageName string
    Percentage float64
}


func readCoverageJsonFromWeb(serviceUrl string) ([]byte){
    //
    // Note:  you need VPN access on for this to work
    //
    res, err := http.Get(serviceUrl)
    check(err)
    defer res.Body.Close()
    body, err := ioutil.ReadAll(res.Body)
    check(err)
    return body
}

//
// Read the Json for the coverage from a cached copy of file 
//
func readCoverageJsonFromFile(serviceCachePath string) ([]byte) {
    contents,err := ioutil.ReadFile(serviceCachePath)
    check(err)
    return contents
}

//
// Give a byte array with all the JSON in it - parses it into the appropriate structs and prints it out
// We assume the list of coverageBranches is in reverse order of date
//
func parseCoverageJson(body []byte, service *HailoService) {
    fmt.Printf("parseCoverageJson \n")
    var  coverageInfos []CoverageInfo
    if err := json.Unmarshal(body, &coverageInfos); err != nil {
        fmt.Printf("error unmarshalling: %v", err)
    }
    foundFirstMasterBranch := false
    //
    // ideally I'd do reverse range but go doesn't support that yet!
    //
    //for _, currentCoverageInfo := range coverageInfos{                
    //
    for i:=len(coverageInfos)-1; i >= 0; i-- {
        fmt.Printf("processing %s\n", coverageInfos[i].Branch)
        if coverageInfos[i].Branch == BranchNameToChoose {
            if !foundFirstMasterBranch {
                fmt.Printf("found MASTER\n")
                for _, currentCoverageInfoItem := range coverageInfos[i].Coverages {
                    fmt.Printf ("\n***appending package: %s \n", currentCoverageInfoItem.PackageName)
                    //
                    // Timsteamps in the data are sorted early to late earliest master package is the earliest data (i.e. it the initial state of coverage)
                    //
                    codePackage := CodePackage{ShortName: currentCoverageInfoItem.PackageName, StartCoveragePercentage: -3, EndCoveragePercentage: currentCoverageInfoItem.Percentage, EndTimestamp : coverageInfos[i].Timestamp}
                    //
                    // see this helpful article about referencing an array in a struct
                    // http://stackoverflow.com/questions/15945030/change-values-while-iterating-in-golang
                    //
                    service.Packages = append (service.Packages, &codePackage) 
                }
                foundFirstMasterBranch = true
            } else {
                for _, currentCoverageInfoItem := range coverageInfos[i].Coverages {
                    fmt.Printf ("\n***updating package: %s\n", currentCoverageInfoItem.PackageName)
                    //
                    // a bit wastefully iterate to find the matching entry in the array of packages and update. 
                    // To make it more wasteful still we do it every time not just once at the end once we've found the last one
                    // a map would be better .:TODO:.
                    // 
                    for _,currentCodePackage := range service.Packages {
                        if currentCodePackage.ShortName == currentCoverageInfoItem.PackageName {
                            //fmt.Printf ("really updating package percentage from: %v to: %v\n", currentCodePackage.StartCoveragePercentage, currentCoverageInfoItem.Percentage)
                            currentCodePackage.StartCoveragePercentage = currentCoverageInfoItem.Percentage
                            currentCodePackage.StartTimestamp = coverageInfos[i].Timestamp
                        } 
                    }
                }
            }
        }
    }
}

//
// Write out the Json for the coverage to a file as a cache for next time   early :Thu, 13 Mar 2014 22:28:46 GMT   late Tue, 11 Mar 2014 20:17:41 GMT
//
func writeCoverageToJson(serviceCachePath string, contents []byte) {
    err := ioutil.WriteFile(serviceCachePath,  contents, 0x666)
    check(err)
}

//
// Checks if there is arecent cached version of the JSOn on disk 
// this is needed as getting the coverage data is slow - and , I imagine, thrashes jenkins a fair bit
//
func cacheFileExistsAndNotTooOld(filePath string) (bool) {
    result := false
    if fileInformation, err := os.Stat(filePath); err != nil {
        if os.IsNotExist(err) {
            result = false
            fmt.Printf("cache file not found:  %s", filePath)
        }
    } else {
        duration := time.Since(fileInformation.ModTime())
        //
        // hopefully this is 7 days
        //
        if duration.Hours() < CacheMaxAgeInDays * HoursInDay  {
            result = true    
            fmt.Printf("cache file %s found and in date range: %f hours old\n", filePath, duration.Hours())
        } else {
            result = false
            fmt.Printf("cache file %s found but NOT in date range: %f hours old\n", filePath, duration.Hours())
        }
    }
    return result
}

//
// Gets the raw coverage data from disk cache if its recent enough or from the 
// web if the cache doesnt exist on disk or is too old
// 
//
func getAndParseCoverageJson(service *HailoService) {
    fileName := CacheDirectoryPath + service.Name + CacheFileExtension
    var body []byte
    if cacheFileExistsAndNotTooOld(fileName) {
        fmt.Printf("read from disk: %s", fileName)
        body = readCoverageJsonFromFile(fileName)
    } else {
        url := UrlForCoveragesStub + service.Name + UrlForCoveragesPostfix
        fmt.Printf("read from URL: %s", url)
        body = readCoverageJsonFromWeb (url)
        //
        // Now cache the Json we got from the URL for next time
        //
        writeCoverageToJson(fileName,body)
    }
    parseCoverageJson(body, service)
}