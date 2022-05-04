package main

import (
	"database/sql"
	"fmt"
	"parsing/products"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/360EntSecGroup-Skylar/excelize"
	"github.com/gocolly/colly"
)

func main() {
	var (
		subCatalog []string
		sumcount   int //счетчик числа записей, используется для записи в excel для определения номера строки
	)
	uniqItems := make(map[string]string)
	exBook := excelize.NewFile()

	db, err := sql.Open("mysql", "mysql:321asd680@tcp(127.0.0.1:3306)/product_scraping")
	products.ErrHandler(err)
	defer db.Close()

	catalog := getCatalogue()

	for _, val := range catalog {
		subCatalog = getSubCatalogue(val, subCatalog)
		fmt.Printf("Собраны ссылки из раздела: %s\n", val)
	}

	for i := 0; i < len(subCatalog); i++ {
		for j := 1; ; j++ {
			if j == 1 {
				stop := getProduct(subCatalog[i], uniqItems, &sumcount, exBook, db)
				if stop {
					break
				}
			} else {
				stop := getProduct(fmt.Sprintf("%s?PAGEN_1=%d", subCatalog[i], j), uniqItems, &sumcount, exBook, db)
				if stop {
					break
				}
			}
			fmt.Printf("Страница %d раздела %s готова\n", j, subCatalog[i])
		}
	}
	fmt.Printf("Загружено товаров: %d\n", sumcount)

	if err := exBook.SaveAs("Globus.xlsx"); err != nil {
		fmt.Println(err)
	}
}

//сбор ссылок из укрупненного каталога
func getCatalogue() []string {
	var catalogue []string
	c := colly.NewCollector()

	c.OnHTML("ul.nav_main__content-list > li", func(e *colly.HTMLElement) {
		element := "https://online.globus.ru" + e.DOM.Find("a").AttrOr("href", "category not found")
		catalogue = append(catalogue, element)
	})

	c.Visit("https://online.globus.ru/catalog")

	return catalogue
}

//сбор ссылок подкаталога для каждой ссылки из укрупненного каталога
func getSubCatalogue(catalogueURL string, subCatalog []string) []string {
	c := colly.NewCollector()

	c.OnHTML("div.section_menu > nav > ul > li", func(e *colly.HTMLElement) {
		subCategory := "https://online.globus.ru" + e.DOM.Find("a").AttrOr("href", "subcategory not found")
		//убираем две категории, т.к. они полностью состоят из дублей
		if !strings.HasSuffix(subCategory, "vygodnye-predlozheniya/") && !strings.HasSuffix(subCategory, "khity-prodazh/") {
			subCatalog = append(subCatalog, subCategory)
		}
	})

	c.Visit(catalogueURL)

	return subCatalog
}

//сбор товаров
func getProduct(subCatalogueURL string, uniqItems map[string]string, sumcount *int, exBook *excelize.File, db *sql.DB) bool {
	var (
		stopScraping bool
		addressURL   string
	)

	c := colly.NewCollector()

	c.OnRequest(func(req *colly.Request) {
		addressURL = req.URL.String()
	})

	/*
		если url реквеста не равен url респонза, то запрашиваемой страницы нет и произошло перенаправление на первую страницу подраздела
		в таком случае парсинг подраздела останавливаем
	*/
	c.OnResponse(func(resp *colly.Response) {
		if addressURL != resp.Request.URL.String() {
			stopScraping = true
		}
	})

	c.OnError(func(resp *colly.Response, err error) {
		if err != nil {
			stopScraping = true
		}
	})

	c.OnHTML(".catalog-section__items.d-row.d-row_ib.js-catalog-section-items > .d-col.d-col_xs_4.d-col_xtr_3.js-catalog-section__item ", func(e *colly.HTMLElement) {
		//проверка переменной остановки парсинга, значение меняем при ошибке или несуществующей странице, см выше
		if stopScraping {
			return
		}

		product := new(products.Products)

		product.Id = e.DOM.AttrOr("product-js-id", "id not found")
		product.Name = e.DOM.Find("div > div > a > span.catalog-section__item__info > span").Text()
		categories := strings.Split(subCatalogueURL, "/")
		product.BigCat = categories[len(categories)-3]
		product.SmallCat = categories[len(categories)-2]
		product.CurrentPrice = strings.ReplaceAll(e.DOM.Find("span.item-price__rub").Text(), " ", "") + "." + e.DOM.Find("span.item-price__kop").Text()
		oldPrice := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(e.DOM.Find("span.item-price__"+
			"old").Text()), " ", "."), "\n", ""), " ", "")
		if oldPrice != "" {
			product.OldPrice = oldPrice
		} else {
			product.OldPrice = "0"
		}
		product.InFactPrice = product.CurrentPrice
		discount := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(e.DOM.Find(".item-sticker.item-sticker-"+
			"catalog--top > span").Text(), "-", ""), "%", ""), "\n", ""), "по карте", "")
		if discount != "" {
			product.Discount = discount
		} else {
			product.Discount = "0"
		}
		product.Unit = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(e.DOM.Find(".item-price__additional.item-price__additional"+
			"--solo").Text(), "за ", ""), "\n", ""))
		product.ProductURL = "https://online.globus.ru" + e.DOM.Find(".catalog-section__item__body.trans > a").AttrOr("href", "URL not found")
		product.Date = time.Now().Format("2006.01.02")
		product.Time = time.Now().Format("15:04:05")

		//если полученный товар уникальный, пишем в эксель и бд
		if product.MapChecking(uniqItems) {
			*sumcount++
			product.ExcelWriting(exBook, *sumcount)
			product.DBWriting(db, "globus")
			fmt.Printf("Собран товар: %s\n", product.Name)
		}
	})

	c.Visit(subCatalogueURL)

	return stopScraping
}
