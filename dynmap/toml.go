package dynmap

import(

)

// Simple parser for TOML 
// https://github.com/mojombo/toml


// Because we are using dynmap we dont need to convert anything, just return the string after verification 
func toVal(str string) (interface{}, bool) {

    s := strings.TrimSpace(str)

    if strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"") {
        //its a string!
        return strings.Trim(s, "\"")
    }

    if strings.EqualFold(s, "true") {
        return true
    }

    if strings.EqualFold(s, "false") {
        return false
    }


    matched, err := regexp.MatchString("[0-9\\.\\-]+", s)
    if err != nil {
        log.Println(err)
    } else if matched {
        //its a number!
        return s
    }

    //test for a date
    // this is a facil check, but hell..
    matched, err = regexp.MatchString("[0-9\\.\\-Z\\:]+", s)
    if err != nil {
        log.Println(err)
    } else if matched {
        //its a date!
        return s
    }


}

func toSlice(str string) ([]interface{}, bool) {

}