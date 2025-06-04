package main

import (
	"fmt"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
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
	for i, divElem := range divElements {
		textContent, err := divElem.Text()
		if err != nil {
			fmt.Printf("Div %d: Error getting text: %v\n", i+1, err)
			continue
		}
		fmt.Printf("Div %d Text:\n%s\n-------------------------------------\n", i+1, textContent)
	}

	// Wait a bit before closing
	time.Sleep(5 * time.Second)
}
