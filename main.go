package main

import (
	"fmt"
	"os"
	"strings"
)

// создание переменных в scan
func scanVariables(structName string, columns []string) string{

	var variables string
 
	for _, column := range columns{
		variables += "&" + structName + "." + column + ", "
	}

	return variables
}

func createPostFields(postFieldsMassive []string) string{
	text := ""

	for _, field := range postFieldsMassive {
		text += fmt.Sprintf("%[1]s := c.PostForm(\"%[1]s\") \n	", strings.ToLower(field) )
	}

	return text
}

func generateVariables(postFieldsMassive []string) (string, string, string){
	first := ""
	second := ""
	third := ""

	for i, field := range postFieldsMassive{
		first += fmt.Sprintf(`\"%[1]s\", `, field)
		second += fmt.Sprintf(`$%[1]d, `, i+1)
		third += fmt.Sprintf(`%[1]s, `, field)
	}

	return first, second, third
}

// сортировка полей для создание дополнительный запросов на выборку по нужному полю или для добавления полей в пост форму
func sortVariables(columns []string) ([]string, []string, []string){

	cleanMassive := ""
	getByMassive := ""
	postFieldsMassive := ""
	

	for _, column := range columns{
		someName := column
		if strings.Contains(column, "*") {
			someName = strings.ReplaceAll(column, "*", "")
			someName = strings.ReplaceAll(someName, "#", "")
			getByMassive += someName + " "
		}

		if strings.Contains(column, "#") {
			someName = strings.ReplaceAll(someName, "#", "")
			postFieldsMassive += someName + " "
		}

		cleanMassive += someName + " "
	}

	return strings.Fields(cleanMassive), strings.Fields(getByMassive),  strings.Fields(postFieldsMassive)
}

func generateString(postFieldsMassive []string) string {
	text := ""

	for i, field := range postFieldsMassive{
		text += field + "=$" + fmt.Sprint(i+1) + " "
	}

	return text
}

func CreatePatchRequest(title string, postFieldsMassive []string) string{

	_, _, third := generateVariables(postFieldsMassive)

	text := fmt.Sprintf(`
func Update%[1]s(c *gin.Context) error {

	id := c.Param("id")

	var %[2]s structs.%[1]s
	

	if err := c.ShouldBindJSON(&%[2]s); err != nil {
		c.String(http.StatusBadGateway, err.Error())
	}
	
	_, err := db.Dbpool.Query("UPDATE Student SET %[3]s, WHERE id=$1", %[4]s )

	if err != nil {
		fmt.Println(err)
	}
	
	c.JSON(http.StatusOK, gin.H{
		"org": %[2]s,
	})
	}
	`, title, strings.ToLower(title), generateString(postFieldsMassive), third)

	return text
}



func PostRequestGenerate(title string, postFieldsMassive []string) string {

	first, second, third := generateVariables(postFieldsMassive)
	text := fmt.Sprintf(`

func Create%[4]s(c *gin.Context) error {
	%[5]s

	_, err := db.Dbpool.Query("INSERT INTO \"%[4]s\"( %[1]s ) VALUES( %[2]s )", %[3]s)

	if err != nil{
		fmt.Println(err)
		return err
}

	c.JSON(200, gin.H{
		"message": "",
	})
}

	`, first, second, third, title, createPostFields(postFieldsMassive))

	return text
}

func GetByRequestGenerate(title string, getByMassive, cleanMassive []string) string {

	text := ""
	for _, name := range getByMassive {
		text += fmt.Sprintf(`

func Get%[1]sBy%[3]s(c *gin.Context) error {
	%[3]s := c.Param("%[3]s")

	rows, err := db.Dbpool.Query("SELECT * FROM \"%[1]s\" WHERE %[3]s=$1", %[3]s) 

	if err != nil{
		fmt.Println(err)
		return err
	}


	var %[2]s structs.%[1]s

	for rows.Next(){
		err = rows.Scan(%[4]s)
		if err != nil{
			fmt.Println(err)
			return err
		}
	}

	c.JSON(200, gin.H{
		"%[1]s": %[2]s,
	})

	return nil
}

		`, title, strings.ToLower(title), name, scanVariables(title, cleanMassive))
	}

	return text
}

func GetRequestGenerate(title string, columns ...string) error {


	cleanMassive, getByMassive, postFieldsMassive := sortVariables(columns)

	getByRequests := ""
	postRequests := ""
	patchsRequests := ""

	if  len(getByMassive) != 0 {
		getByRequests = GetByRequestGenerate(title, getByMassive, cleanMassive)
	}

	if len(postFieldsMassive) != 0 {
		postRequests = PostRequestGenerate(title, postFieldsMassive)
		patchsRequests = CreatePatchRequest(title, postFieldsMassive)
	}

	text := fmt.Sprintf(`package controllers

import (
	"github.com/gin-gonic/gin"
	"path to your package database/db"
	"path to your package structsё/structs"
)

func Get%[1]s(c *gin.Context) error {

	rows, err := db.Dbpool.Query("SELECT * FROM \"%[1]s\"") 

	if err != nil{
		fmt.Println(err)
		return err
	}

	var %[2]s_list []structs.%[1]s
	var %[2]s structs.%[1]s

	for rows.Next(){
		err = rows.Scan(%[3]s)
		%[2]s_list = append(%[2]s_list, %[2]s)
		if err != nil{
			fmt.Println(err)
			return err
		}
	}

	c.JSON(200, gin.H{
		"%[2]s's": %[2]s_list,
	})

	return nil
}

%[4]s

%[5]s

%[6]s

func Delete%[1]s(c *gin.Context) error{
	id := c.Param("id")

	_, err := db.Dbpool.Query("DELETE FROM \"%[1]s\" WHERE id=$1", id)

	if err != nil{
		fmt.Println(err)
		return err
	}

	c.JSON(200, gin.H{
		"message": "",
	})

	return nil
}
	
	`, title,
	strings.ToLower(title), 
	scanVariables(title, cleanMassive),
	getByRequests,
	postRequests,
	patchsRequests,
)

	file, err := os.Create(fmt.Sprintf(`%[1]s.controllers.go`, title))
	if err != nil {
		// fmt.Println(err)
		return err
	}
	fmt.Fprint(file, text)
	file.Close()

	return nil
}

func main(){
	GetRequestGenerate("User", "*#Id", "#Name", "#SureName", "#GroupId")
}
