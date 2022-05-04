package products

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/360EntSecGroup-Skylar/excelize"
	_ "github.com/go-sql-driver/mysql"
)

type Products struct {
	Id           string
	BigCat       string
	SmallCat     string
	Name         string
	CurrentPrice string
	OldPrice     string
	InFactPrice  string
	Discount     string
	Unit         string
	ProductURL   string
	Date         string
	Time         string
}

//формирование массива из структуры для загрузки в excel
func (pds *Products) Compound() (cmp []string) {
	return []string{pds.Id, pds.BigCat, pds.SmallCat, pds.Name, pds.OldPrice, pds.Discount, pds.CurrentPrice, pds.InFactPrice, pds.Unit,
		pds.ProductURL, pds.Date, pds.Time}
}

//запись в excel файл
func (pds *Products) ExcelWriting(book *excelize.File, sumcount int) {
	var r rune = 64
	b := pds.Compound()
	for i := 1; i < len(b)+1; i++ {
		book.SetCellValue("Sheet1", fmt.Sprintf("%c"+strconv.Itoa(sumcount+1), r+int32(i)), b[i-1])
	}
}

//мапа для проверки уникальности объектов при парсинге
func (pds *Products) MapChecking(uniqItems map[string]string) (ok bool) {
	if _, ok := (uniqItems)[pds.Id]; ok {
		return !ok
	} else {
		(uniqItems)[pds.Id] = pds.Name
	}
	return !ok
}

//запись данных в базу
func (pds *Products) DBWriting(db *sql.DB, DBName string) {
	var (
		newOldPr, newCurPr, newInFPr = 0.00, 0.00, 0.00
		newDiscount                  = 0
	)
	newId, err := strconv.Atoi(pds.Id)
	ErrHandler(err)
	if pds.OldPrice != "" {
		newOldPr, err = strconv.ParseFloat(pds.OldPrice, 32)
		ErrHandler(err)
	}
	if pds.CurrentPrice != "" {
		newCurPr, err = strconv.ParseFloat(pds.CurrentPrice, 32)
		ErrHandler(err)
	}
	if pds.InFactPrice != "" {
		newInFPr, err = strconv.ParseFloat(pds.InFactPrice, 32)
		ErrHandler(err)
	}
	if pds.Discount != "" {
		newDiscount, err = strconv.Atoi(strings.Trim(pds.Discount, "-"))
		ErrHandler(err)
	}

	insert, err := db.Query(fmt.Sprintf("INSERT INTO %s (`id`, `big_category`, `small_category`, `product_name`, "+
		"`old_price`, `current_price`, `in_fact_price`, `discount`, `unit`, `url`, `date`, `time`) VALUES('%d', '%s', '%s', '%s', '%f', '%f', "+
		"'%f', '%d', '%s', '%s', '%s', '%s')", DBName, newId, pds.BigCat, pds.SmallCat, strings.ReplaceAll(pds.Name, "'", ""), newOldPr,
		newCurPr, newInFPr, newDiscount, pds.Unit, pds.ProductURL, pds.Date, pds.Time))
	ErrHandler(err)
	insert.Close()
}

func ErrHandler(err error) {
	if err != nil {
		fmt.Println(err)
	}
}
