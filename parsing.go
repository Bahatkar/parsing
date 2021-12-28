package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/360EntSecGroup-Skylar/excelize"
	"github.com/gocolly/colly"
)

type Products struct {
	id       string
	bigCat   string
	smallCat string
	name     string
	oldPrice string
	discount string
	newPrice string
	unit     string
	url      string
	date     string
	time     string
}

func main() {
	var itemCount, sumcount int
	uniqItems := make(map[string]string)
	catalogue := getCatalogue()

	f := excelize.NewFile()

	for j := 0; j < len(catalogue); j++ {
		for i := 1; ; i++ {
			if i == 1 {
				itemCount = gethttp(catalogue[j], f, uniqItems, &sumcount)
				if itemCount < 1 {
					break
				}
			} else {
				itemCount = gethttp(fmt.Sprintf("%s?page=%d", catalogue[j], i), f, uniqItems, &sumcount)
				if itemCount < 1 {
					break
				}
			}
			fmt.Printf("Страница %d раздела %s готова\n", i, catalogue[j])
		}
	}
	fmt.Printf("Загружено товаров: %d\n", sumcount)

	if err := f.SaveAs("Perik.xlsx"); err != nil {
		fmt.Println(err)
	}
}

func gethttp(uurl string, f *excelize.File, uniqItems map[string]string, sumCount *int) (itemCount int) {
	var (
		reqUrl    string
		stopScrap bool
	)
	itemCount = 0

	c := colly.NewCollector()

	c.OnRequest(func(h *colly.Request) {
		reqUrl = h.URL.String()
	})

	c.OnResponse(func(g *colly.Response) {
		if reqUrl != g.Request.URL.String() {
			stopScrap = true
			return
		}
	})

	c.OnHTML(".js-catalog-product", func(e *colly.HTMLElement) {
		pds := new(Products)

		if stopScrap {
			return
		}

		pds.id = strings.TrimSpace(e.DOM.Find(".xf-product").AttrOr("data-id", ""))
		pds.bigCat = strings.TrimSpace((e.DOM.Find(".xf-product").AttrOr("data-owox-portal-name", "")))
		pds.smallCat = strings.TrimSpace((e.DOM.Find(".xf-product").AttrOr("data-owox-category-name", "")))
		pds.name = strings.TrimSpace(e.DOM.Find(".xf-product-title__link").Text())
		if pds.name != "" {
			itemCount++
		}
		pds.discount = strings.TrimSpace(e.DOM.Find(".js-calculated-discount").Text())
		if pds.discount != "" {
			pds.discount += "%"
		}
		pds.newPrice = strings.TrimSpace(e.DOM.Find("div.xf-price.xf-product-cost__current.js-product__cost").AttrOr("data-cost", ""))
		pds.oldPrice = strings.TrimSpace(e.DOM.Find("div.xf-price.xf-product-cost__prev.js-product__old-cost").AttrOr("data-cost", ""))
		pds.unit = strings.TrimSpace(e.DOM.Find(".js-fraction-text").Text())
		pds.url = strings.TrimSpace(e.DOM.Find(".xf-product").AttrOr("data-product-card-url", ""))
		if pds.bigCat == "Зоотовары" {
			pds.url = fmt.Sprintf("https://zoo.vprok.ru" + pds.url)
		} else {
			pds.url = fmt.Sprintf("https://www.vprok.ru" + pds.url)
		}
		pds.date = time.Now().Format("02.01.2006")
		pds.time = time.Now().Format("15:04:05")

		if pds.mapChecking(uniqItems) {
			*sumCount++
			excelWriting(f, pds, *sumCount)
			fmt.Printf("Собран товар: %v\n", pds.name)
		}
	})

	c.Visit(uurl)

	return
}

func getCatalogue() []string {
	var catalogue = make([]string, 0)
	c := colly.NewCollector()

	c.OnHTML("div.xf-catalog-categories__col._right li", func(e *colly.HTMLElement) {
		href, ok := e.DOM.Find(".xf-catalog-categories__link").Attr("href")
		if ok && strings.TrimSpace(href) == "/catalog/1308/tovary-dlya-jivotnyh" {
			catalogue = append(catalogue, fmt.Sprintf("https://zoo.vprok.ru"+strings.TrimSpace(href)))
		} else if ok {
			catalogue = append(catalogue, fmt.Sprintf("https://www.vprok.ru"+strings.TrimSpace(href)))
		}
	})

	c.Visit("https://www.vprok.ru/catalog")

	return catalogue
}

func (pds *Products) compound() (cmp []string) {
	return []string{pds.id, pds.bigCat, pds.smallCat, pds.name, pds.oldPrice, pds.discount, pds.newPrice, pds.unit, pds.url,
		pds.date, pds.time}
}

func excelWriting(book *excelize.File, pds *Products, sumcount int) {
	var r rune = 64
	b := pds.compound()
	for i := 1; i < len(b)+1; i++ {
		book.SetCellValue("Sheet1", fmt.Sprintf("%c"+strconv.Itoa(sumcount+1), r+int32(i)), b[i-1])
	}
}

func (pds *Products) mapChecking(uniqItems map[string]string) (ok bool) {
	if _, ok := (uniqItems)[pds.id]; ok {
		return !ok
	} else {
		(uniqItems)[pds.id] = pds.name
	}
	return !ok
}
