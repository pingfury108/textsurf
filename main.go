package main

import (
	"fmt"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

func main() {
	// Launch a new browser
	url := launcher.New().
		Headless(false). // Set to true for headless mode
		MustLaunch()

	// Create a new browser and page
	browser := rod.New().ControlURL(url).MustConnect()
	defer browser.MustClose()

	page := browser.MustPage("https://m.douyinhanyu.com/search?")

	// Wait for the page to load
	page.MustWaitStable()

	page.MustElement("body > div > div > div > header.header_nsm1M-.bg-template-header > div.transparentWrapper_f\\+x1Vm > div > div.fixed_lUl90b.content_MgtYFU > div > div.title_3WacpN.flex_sGoqor.hanyu-navbar-title > div > div > input").MustInput("岳阳楼记")

	page.MustElement("body > div > div > div > header.header_nsm1M-.bg-template-header > div.transparentWrapper_f\\+x1Vm > div > div.fixed_lUl90b.content_MgtYFU > div > div.right_1TKXtq.hanyu-navbar-right > aside").MustClick()

	// Wait for the search results to load
	time.Sleep(2 * time.Second)
	page.MustWaitStable()

	// Get the content from the three div children
	fmt.Println("Attempting to extract text from three div children of main element:")
	divsSelector := "body > div > div > div > main > div.transparentWrapper_f\\+x1Vm > div > div > div > main > div"
	fmt.Println("Using selector for divs:", divsSelector)

	divElements, err := page.Elements(divsSelector)
	if err != nil {
		fmt.Printf("Error finding div elements with selector '%s': %v\n", divsSelector, err)
		// Print a snippet of the page HTML for debugging purposes if elements are not found:
		html := page.MustHTML()
		fmt.Println("\nPage HTML structure (first 1000 chars to help debug selector):")
		if len(html) > 1000 {
			fmt.Println(html[:1000] + "...")
		} else {
			fmt.Println(html)
		}
		return
	}

	if len(divElements) == 0 {
		fmt.Printf("No div elements found with selector '%s'\n", divsSelector)
		// Print a snippet of the page HTML for debugging purposes if elements are not found:
		html := page.MustHTML()
		fmt.Println("\nPage HTML structure (first 1000 chars to help debug selector):")
		if len(html) > 1000 {
			fmt.Println(html[:1000] + "...")
		} else {
			fmt.Println(html)
		}
		return
	}

	fmt.Println("\nText content from the div elements:")
	fmt.Println("-------------------------------------")
	
	// Click on the first element
	if len(divElements) > 0 {
		fmt.Println("Clicking on the first div element...")
		err := divElements[0].Click(proto.InputMouseButtonLeft, 1)
		if err != nil {
			fmt.Printf("Error clicking first element: %v\n", err)
		} else {
			fmt.Println("Successfully clicked on the first element!")
			
			// Wait for any content to load after clicking
			time.Sleep(2 * time.Second)
			page.MustWaitStable()
			
			// Print the content accessed after clicking
			fmt.Println("\nContent after clicking the first element:")
			fmt.Println("=========================================")
			
			// Extract text content from the page after clicking
			pageText, err := page.MustElement("body").Text()
			if err != nil {
				fmt.Printf("Error getting page text: %v\n", err)
			} else {
				fmt.Printf("Page text content after clicking:\n%s\n", pageText)
			}
			
			// Also try to get text from the clicked element
			textContent, err := divElements[0].Text()
			if err != nil {
				fmt.Printf("Error getting text from clicked element: %v\n", err)
			} else {
				fmt.Printf("\nText from clicked element:\n%s\n", textContent)
			}
		}
	}

	// Wait a bit before closing
	time.Sleep(5 * time.Second)
}
