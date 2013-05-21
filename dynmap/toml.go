package dynmap

import(
    "strings"
    "regexp"
    "log"
)

// Simple parser for TOML 
// https://github.com/mojombo/toml


func ParseTOML(str string) (*DynMap, error) {


}

type toml struct {
    r *bufio.Reader
    position int
    bracketCount int
    output *DynMap
    key string
}

func (this *toml) parseNext() error {
    str, err := readln(this.r)

    if err := nil {
        return err
    }

    str = strings.TrimSpace(str)
    if str == "" {
        //blank line
        return nil
    }

    if strings.HasPrefix(str, "#") {
        // comment line
        return nil
    }

    if strings.HasPrefix(str, "[") {
        //change the key
        str = strings.TrimPrefix(str, "[")
        str = strings.TrimSuffix(str, "]")
        // allways have a trailing dot
        this.key = fmt.Sprintf("%s.", str)
        return nil
    }

    //we are parsing a key value!
    tmp := strings.SplitN(str, "=", 2)
    if len(tmp) != 2 {
        return fmt.Errorf("Error on line: %s, no equals sign!", str)
    }
    key = strings.TrimSpace(tmp[0])
    value = strings.TrimSpace(tmp[1])

    if !strings.HasPrefix(value, "[") {
        //its not an array
        v, ok := toVal(value)
        if !ok {
            return fmt.Errorf("Error on line: %s, unable to parse value %s", str, value)
        }
        this.output.PutWithDot(fmt.Sprintf("%s%s",this.key,key), v)
        return nil
    }

    //ok parse the damn array

    //arrays can contain multiple lines
    // so we count the opening and closing brackets.
    for strings.Count(str, "[") != strings.Count(str, "]") {
        ln, err := readln(this.r)
        if err != nil {
            return err
        }
        str = fmt.Sprintf("%s %s", str, ln)
    }



}

func toSlice(str string, pos int) ([]interface{}, bool, int) {
    ret := make([]interface{}, 0)
    //assume str[pos] == "["

    cur := ""
    for i := pos+1; i < len(str); {
        
        r, width := utf8.DecodeRuneInString(str[i:])
        
        if r == '[' {
            //start another array
            v, ok, pos := toSlice(str, i)
            if !ok {
                return ret, false, i
            }
            ret = append(ret, v)
            i = pos
            continue
        }
 
        i += width


    }    


}



func readln(r *bufio.Reader) (string, error) {
  var (isPrefix bool = true
       err error = nil
       line, ln []byte
      )
  for isPrefix && err == nil {
      line, isPrefix, err = r.ReadLine()
      ln = append(ln, line...)
  }
  return string(ln),err
}

//return the quoted string (minus the quotes)
// return the ending position in the string (closing quote + 1)
func toString(str string) (string, bool, int) {
    str = strings.Trim(s, "\"")


}

// Because we are using dynmap we dont need to convert anything, just return the string after verification 
func toVal(str string) (interface{}, bool, int) {

    s := strings.TrimSpace(str)

    //TODO Deal with the damn comments!

    if strings.HasPrefix(s, "\"") {
        //its a string!



        return strings.Trim(s, "\""), true
    }

    if strings.EqualFold(s, "true") {
        return true, true
    }

    if strings.EqualFold(s, "false") {
        return false, true
    }


    matched, err := regexp.MatchString("[0-9\\.\\-]+", s)
    if err != nil {
        log.Println(err)
    } else if matched {
        //its a number!
        return s, true
    }

    //test for a date
    // this is a facil check, but hell..
    matched, err = regexp.MatchString("[0-9\\.\\-Z\\:]+", s)
    if err != nil {
        log.Println(err)
    } else if matched {
        //its a date!
        return s, true
    }

    return "", false
}

