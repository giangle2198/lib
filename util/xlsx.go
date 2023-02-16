package util

import (
	"fmt"
	"reflect"
	"time"

	"github.com/xuri/excelize/v2"
)

type (
	FileDTO struct {
		TitleName     string
		FileName      string
		Base64File    string
		Status        string
		FileChunk     int
		TransactionID string
		FileType      string
		DomainUser    string
		HeaderFile    []string
		Style         int
		LocalTimeZone uint32
	}
)

func WriteRequestDataXlsxExcelize(streamWriter *excelize.StreamWriter,
	input interface{}, indexRow int, style int) (newIndexRow int, err error) {
	typeOf := reflect.TypeOf(input)
	valueOf := reflect.ValueOf(input)
	if typeOf.Kind() == reflect.Ptr {
		valueOf = valueOf.Elem()
		typeOf = typeOf.Elem()
	}
	numberFiled := typeOf.NumField()
	newIndexRow = indexRow
	for i := 0; i < numberFiled; i++ {
		var items []interface{}
		field := typeOf.Field(i)
		valueClassifier := valueOf.Field(i)
		items = append(items, excelize.Cell{Value: field.Name}, excelize.Cell{Value: valueClassifier})
		newIndexRow++
		err = streamWriter.SetRow(fmt.Sprintf("A%v", newIndexRow), items)
		if err != nil {
			return newIndexRow, err
		}
	}
	return newIndexRow, nil
}

func WriteTitleXlsxExcelize(streamWriter *excelize.StreamWriter,
	req *FileDTO) int {
	rowIndex := 1
	var axis string
	axis = fmt.Sprintf("A%v", rowIndex)
	styleFileName := []interface{}{excelize.Cell{StyleID: req.Style, Value: req.TitleName}}
	_ = streamWriter.SetRow(axis, styleFileName)
	rowIndex++
	userCreate := fmt.Sprintf("Created by: %v", req.DomainUser)
	axis = fmt.Sprintf("A%v", rowIndex)
	itemValue := []interface{}{excelize.Cell{StyleID: req.Style, Value: userCreate}}
	_ = streamWriter.SetRow(axis, itemValue)
	rowIndex++
	timeCreateReports := fmt.Sprintf("Created Time: %v", time.Now().Add(
		time.Hour*time.Duration(req.LocalTimeZone)).Format("2006/01/02 15:04:05"))
	axis = fmt.Sprintf("A%v", rowIndex)
	valye := []interface{}{excelize.Cell{StyleID: req.Style, Value: timeCreateReports}}
	_ = streamWriter.SetRow(axis, valye)
	return rowIndex
}

func WriteHeaderXlsxExcelize(streamWriter *excelize.StreamWriter, req *FileDTO, indexRow int) int {
	newIndexRow := indexRow + 1
	newHeader := make([]interface{}, len(req.HeaderFile))
	for i, v := range req.HeaderFile {
		newHeader[i] = excelize.Cell{StyleID: req.Style, Value: fmt.Sprint(v)}
	}
	_ = streamWriter.SetRow(fmt.Sprintf("A%v", newIndexRow), newHeader)
	return newIndexRow
}

func WriteValueXlsxExcelize(streamWriter *excelize.StreamWriter,
	cellValues *[]interface{}, rowIndex int) (num int, err error) {
	newIndexRow := rowIndex
	var item []interface{}
	for i := 0; i < len(*cellValues); i++ {
		item = (*cellValues)[i].([]interface{})
		newIndexRow++
		err = streamWriter.SetRow(fmt.Sprintf("A%v", newIndexRow), item)
		if err != nil {
			return newIndexRow, err
		}
	}
	return newIndexRow, nil
}

func setValueByType(dataValue reflect.Value, dataType reflect.Type) []interface{} {
	var items []interface{}
	switch dataType.Kind() {
	case reflect.Ptr:
		dataValueOf := dataValue.Elem()
		// dataValueIntfOf := reflect.Indirect(reflect.ValueOf(itemIntf))
		typeOf := dataType.Elem()
		items = setValueByType(dataValueOf, typeOf)
	case reflect.Map:
		for _, key := range dataValue.MapKeys() {
			switch dataValue.MapIndex(key).Kind() {
			case reflect.Float64:
				items = append(items, excelize.Cell{Value: dataValue.MapIndex(key).Float()})
			case reflect.String:
				items = append(items, excelize.Cell{Value: dataValue.MapIndex(key).String()})
			default:
				items = append(items, excelize.Cell{Value: dataValue.MapIndex(key).String()})
			}
		}

	case reflect.Struct:
		for i := 0; i < dataValue.NumField(); i++ {
			switch dataValue.Field(i).Kind() {
			case reflect.Float64:
				items = append(items, excelize.Cell{Value: dataValue.Field(i).Float()})
			case reflect.String:
				items = append(items, excelize.Cell{Value: dataValue.Field(i).String()})
			case reflect.Uint32:
				items = append(items, excelize.Cell{Value: dataValue.Field(i).Uint()})
			default:
				items = append(items, excelize.Cell{Value: dataValue.Field(i).String()})
			}
		}
	case reflect.Slice:
		for i := 0; i < dataValue.Len(); i++ {
			switch dataValue.Index(i).Kind() {
			case reflect.Float64:
				items = append(items, excelize.Cell{Value: dataValue.Index(i).Float()})
			case reflect.String:
				items = append(items, excelize.Cell{Value: dataValue.Index(i).String()})
			default:
				items = append(items, excelize.Cell{Value: dataValue.Index(i).String()})
			}
		}
	}

	return items
}

func WriteValueXLSXByType(streamWriter *excelize.StreamWriter,
	cellValues *[]interface{}, rowIndex int) (num int, err error) {

	newIndexRow := rowIndex + 1
	for _, cardDetailItem := range *cellValues {
		var items []interface{}
		dataValueOf := reflect.ValueOf(cardDetailItem)
		typeOf := reflect.TypeOf(cardDetailItem)
		// switch typeOf.Kind() {
		// case reflect.Ptr:
		// 	itemIntf := dataValueOf.Elem()
		// 	dataValueIntfOf := reflect.Indirect(reflect.ValueOf(itemIntf))
		// 	typeIntfOf := reflect.TypeOf(itemIntf)
		// 	items = setValueByType(dataValueIntfOf, typeIntfOf)
		// }
		items = setValueByType(dataValueOf, typeOf)
		err = streamWriter.SetRow(fmt.Sprintf("A%v", newIndexRow), items)
		if err != nil {
			return newIndexRow, err
		}
		newIndexRow++
	}
	return newIndexRow, nil
}
