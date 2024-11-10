package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

const memoPath = "numbers.txt"
var memo []int

func ParseMemo() ([]int, error) {
	data, err := os.ReadFile(memoPath)
	if err != nil {
		log.Printf("error while reading memo file: %s\n", err)
		return nil, err
	}
	
	videoNumbers := []int{}

	splits := strings.Split(string(data[:]), "\n")
	for _, str := range splits {
		if str == "" {
			continue
		}
		num, err := strconv.Atoi(str)
		if err != nil {
			return nil, err
		}
		videoNumbers = append(videoNumbers, num)
	}
	memo = videoNumbers

	return videoNumbers, nil
}

func AddMemo(videoNumber int) error {
	file, err := os.OpenFile(memoPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(fmt.Sprintf("%d\n", videoNumber))
	if err != nil {
		return err
	}

	memo = append(memo, videoNumber)

	return nil
}

func CheckMemo(videoNumber int) bool {
	exists := false
	if memo == nil {
		ParseMemo()
	}

	for _, num := range memo {
		if num == videoNumber {
			exists = true
		}
	}

	return exists
}
