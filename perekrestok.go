package main

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"parsing/products"

	_ "github.com/go-sql-driver/mysql"

	"github.com/360EntSecGroup-Skylar/excelize"
	"github.com/gocolly/colly"
)

func main() {
	var itemCount, sumcount int //itemcount хранит кол-во собранных товаров со страницы, sumcount - общее число собранных товаров
	uniqItems := make(map[string]string)
	catalogue := getCatalogue()

	f := excelize.NewFile()

	db, err := sql.Open("mysql", "mysql:321asd680@tcp(127.0.0.1:3306)/product_scraping")
	products.ErrHandler(err)
	defer db.Close()

	for j := 0; j < len(catalogue); j++ {
		for i := 1; ; i++ {
			//у первой страницы нет номера "?page=1", т.ч. под нее создаю отдельную ветку
			if i == 1 {
				itemCount = getProduct(catalogue[j], f, uniqItems, &sumcount, db)
				if itemCount < 1 { //если со страницы собрано 0 товаров, то прекращаем парсить раздел и идем к след.
					break
				}
			} else {
				itemCount = getProduct(fmt.Sprintf("%s?page=%d", catalogue[j], i), f, uniqItems, &sumcount, db)
				if itemCount < 1 { //если со страницы собрано 0 товаров, то прекращаем парсить раздел и идем к след.
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

func getProduct(uurl string, f *excelize.File, uniqItems map[string]string, sumCount *int, db *sql.DB) (itemCount int) {
	var (
		reqUrl    string
		stopScrap bool
	)
	itemCount = 0

	c := colly.NewCollector()

	c.OnRequest(func(h *colly.Request) {
		reqUrl = h.URL.String()
	})

	/*
		если url реквеста не равен url респонза, то запрашиваемой страницы нет и произошло перенаправление на первую страницу подраздела
		в таком случае парсинг подраздела останавливаем. В Перекрестке таких проблем нет, это для подстраховки
	*/
	c.OnResponse(func(g *colly.Response) {
		if reqUrl != g.Request.URL.String() {
			stopScrap = true
			return
		}
	})

	c.OnHTML(".js-catalog-product", func(e *colly.HTMLElement) {
		pds := new(products.Products)

		//проверка переменной остановки парсинга, значение меняем при ошибке или несуществующей странице
		if stopScrap {
			return
		}

		pds.Id = strings.TrimSpace(e.DOM.Find(".xf-product").AttrOr("data-id", ""))
		pds.BigCat = strings.TrimSpace((e.DOM.Find(".xf-product").AttrOr("data-owox-portal-name", "")))
		pds.SmallCat = strings.TrimSpace((e.DOM.Find(".xf-product").AttrOr("data-owox-category-name", "")))
		pds.Name = strings.TrimSpace(e.DOM.Find(".xf-product-title__link").Text())
		/*
			если на странице есть название товара, то увеличиваем счетчик числа товаров на странице. Если этот счетчик вернется с нулем,
			значит страница пустая и парсинг раздела останавливается
		*/
		if pds.Name != "" {
			itemCount++
		}
		pds.Discount = strings.TrimSpace(e.DOM.Find(".js-calculated-discount").Text())
		pds.CurrentPrice = strings.TrimSpace(e.DOM.Find("div.xf-price.xf-product-cost__current.js-product__cost").AttrOr("data-cost", ""))
		pds.InFactPrice = pds.CurrentPrice
		pds.OldPrice = strings.TrimSpace(e.DOM.Find("div.xf-price.xf-product-cost__prev.js-product__old-cost").AttrOr("data-cost", ""))
		pds.Unit = strings.TrimSpace(e.DOM.Find(".js-fraction-text").Text())
		pds.ProductURL = strings.TrimSpace(e.DOM.Find(".xf-product").AttrOr("data-product-card-url", ""))
		if pds.BigCat == "Зоотовары" {
			pds.ProductURL = fmt.Sprintf("https://zoo.vprok.ru" + pds.ProductURL)
		} else {
			pds.ProductURL = fmt.Sprintf("https://www.vprok.ru" + pds.ProductURL)
		}
		pds.Date = time.Now().Format("2006.01.02")
		pds.Time = time.Now().Format("15:04:05")

		//если товар уникальный, то пишем его в базу и excel
		if pds.MapChecking(uniqItems) {
			*sumCount++
			pds.ExcelWriting(f, *sumCount)
			pds.DBWriting(db, "perekrestok")
			fmt.Printf("Собран товар: %v\n", pds.Name)
		}
	})

	c.Visit(uurl)

	return
}

//сбор ссылок каталога
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
