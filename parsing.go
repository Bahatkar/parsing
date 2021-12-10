package main

import (
	"fmt"
	"strings"

	"github.com/gocolly/colly"
)

func main() {
	//gethttp("https://www.vprok.ru/catalog/4019/sladosti-i-sneki")
	gethttp("https://www.vprok.ru/catalog/1304/ryba-i-moreprodukty")
}

func gethttp(url string) {
	c := colly.NewCollector()

	c.OnHTML("main > div> div > ul li", func(e *colly.HTMLElement) {
		name := strings.TrimSpace(e.DOM.Find("div.xf-product__title").Text())
		discount := strings.TrimSpace(e.DOM.Find("div.xf-product__cost > div.xf-product-cost__old-price > p.js-calculated-discount").Text())
		if discount != "" {
			discount += "%"
		}
		newPrice := strings.TrimSpace(e.DOM.Find("div.xf-product__cost > div.xf-price > span.xf-price__rouble").Text() +
			e.DOM.Find("div.xf-product__cost > div.xf-price > span.xf-price__penny").Text())
		oldPrice := strings.TrimSpace(e.DOM.Find("div.xf-product__cost > div.xf-product-cost__old-price > div.xf-price").Text())
		unit := strings.TrimSpace(e.DOM.Find("div.xf-product__cost > div.xf-price > span.xf-price__unit").Text())
		fmt.Println("Наименование: " + name)
		fmt.Println("Цена без скидки: " + oldPrice)
		fmt.Println("Размер скидки: " + discount)
		fmt.Println("Цена со скидкой: " + newPrice)
		fmt.Println("Количество: " + unit)
	})

	c.Visit(url)
}
