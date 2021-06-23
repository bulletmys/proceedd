package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
)

type Server struct {
	Name   string
	Cpu    []int
	Mem    []int
	Weight []int
}

type Srv struct {
	ip  string
	Cpu int
	Mem int
}

type SrvWeight struct {
	ip     string
	weight int
}

var iptoname = map[string]*Server{
	"151.248.116.244:5000": {Name: "Medium 2"},
	"80.78.244.210:5000":   {Name: "Medium"},
	"151.248.117.144:5000": {Name: "Low"},
	"31.31.201.127:5000":   {Name: "Low Plus"},
	"89.108.71.153:5000":   {Name: "High"},
}

func main() {

	file, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	re, err := regexp.Compile("Host: (?P<ip>(\\d{1,3}[.:]){4}\\d+), Data: (?P<cpu>\\d+):(?P<mem>\\d+)")
	if err != nil {
		log.Fatal(err)
	}
	reWeight, err := regexp.Compile("(?P<ip>(\\d{1,3}[.:]){4}\\d+) prevW: (?P<w1>\\d+). actualW: (?P<w2>\\d+)")
	if err != nil {
		log.Fatal(err)
	}

	s := bufio.NewScanner(file)
	for s.Scan() {
		groupNames := re.SubexpNames()

		for _, match := range re.FindAllStringSubmatch(s.Text(), -1) {
			var srv Srv

			for groupIdx, group := range match {
				name := groupNames[groupIdx]
				if name == "ip" {
					srv.ip = group
				}
				if name == "cpu" {
					i, err := strconv.Atoi(group)
					if err != nil {
						log.Fatal(err)
					}
					srv.Cpu = i
				}
				if name == "mem" {
					i, err := strconv.Atoi(group)
					if err != nil {
						log.Fatal(err)
					}
					srv.Mem = i
				}
			}
			if srv.Cpu == 0 {
				continue
			}

			server := iptoname[srv.ip]

			server.Cpu = append(server.Cpu, srv.Cpu)
			//server.Mem = append(server.Mem, srv.Mem)
		}

		groupNames = reWeight.SubexpNames()


		for _, match := range reWeight.FindAllStringSubmatch(s.Text(), -1) {
			var srv SrvWeight

			for groupIdx, group := range match {
				name := groupNames[groupIdx]
				if name == "ip" {
					srv.ip = group
				}
				if name == "w1" {
					i, err := strconv.Atoi(group)
					if err != nil {
						log.Fatal(err)
					}
					srv.weight = i
				}
			}
			server := iptoname[srv.ip]
			server.Weight = append(server.Weight, srv.weight)
		}
	}

	if err := s.Err(); err != nil {
		log.Fatal(err)
	}

	//lowR := 3
	//highR := 62
	//
	//for _, val := range iptoname {
	//	fmt.Println(val.Name)
	//	for i, elem := range val.Cpu {
	//		if i < lowR {
	//			continue
	//		}
	//		if i > highR {
	//			continue
	//		}
	//		fmt.Println(elem)
	//	}
	//	//for i, elem := range val.Mem {
	//	//	if i < skip {
	//	//		continue
	//	//	}
	//	//	fmt.Println(elem)
	//	//}
	//}

	lowRw := 0
	highRw := 59

	for _, val := range iptoname {
		fmt.Println(val.Name)
		for i, elem := range val.Weight {
			if i < lowRw {
				continue
			}
			if i > highRw {
				continue
			}
			fmt.Println(elem)
		}
	}
}
