package bencode_test

import (
	"bytes"
	"fmt"
	"github.com/miniriley2012/torrent/bencode"
	"testing"
	"time"
)

type Person struct {
	FirstName string
	LastName  string
	Age       int
}

type Time struct {
	time.Time
}

func (t Time) MarshalBencode() ([]byte, error) {
	return bencode.Marshal(t.Time.Unix())
}

func (t *Time) UnmarshalBencode(data []byte) (int, error) {
	var i int64
	n, err := bencode.Unmarshal(data, &i)
	t.Time = time.Unix(i, 0)
	return n, err
}

type TestStruct struct {
	People      []Person
	Thing       []byte `bencode:"thing"`
	DoubleSlice [][]int
	Len         uint
	Time        Time
}

func TestMarshalMap(t *testing.T) {
	m := map[string]interface{}{
		"a":     1,
		"b":     100,
		"hello": []string{"world", "or", "John"},
		"map": map[string]interface{}{
			"something": map[string]string{
				"very": "complex",
			},
		},
		"person": Person{
			FirstName: "John",
			LastName:  "Doe",
			Age:       32,
		},
	}
	b, err := bencode.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(string(b))
}

func TestUnmarshalMap(t *testing.T) {
	m := map[string]interface{}{}
	_, err := bencode.Unmarshal([]byte("d3:keyi10e5:value9:something5:thingli11ei12ei13eee"), &m)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(m)
}

func TestEqualMap(t *testing.T) {
	m := map[string]interface{}{
		"a":     1,
		"b":     100,
		"hello": []string{"world", "or", "John"},
		"map": map[string]interface{}{
			"something": map[string]string{
				"very": "complex",
			},
		},
		"person": Person{
			FirstName: "John",
			LastName:  "Doe",
			Age:       32,
		},
	}

	b, err := bencode.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("Marshaled:", string(b))

	m2 := map[string]interface{}{}

	if _, err = bencode.Unmarshal(b, &m2); err != nil {
		t.Fatal(err)
	}

	b2, _ := bencode.Marshal(m2)
	fmt.Println("Marshaled again:", string(b2))
	fmt.Println("Equal:", bytes.Equal(b, b2))
}

func TestMarshalStruct(t *testing.T) {
	b, err := bencode.Marshal(&testStruct)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(b))
}

func TestUnmarshalStruct(t *testing.T) {
	b, _ := bencode.Marshal(&testStruct)

	var ts TestStruct
	if _, err := bencode.Unmarshal(b, &ts); err != nil {
		t.Fatal(err)
	}
}

func TestMarshalBadType(t *testing.T) {
	_, err := bencode.Marshal(complex(0, 0))
	if err == nil {
		t.Fatal("No error")
	}
	fmt.Println(err)
}

func TestMarshalBadList(t *testing.T) {
	_, err := bencode.Marshal([]complex128{0})
	if err == nil {
		t.Fatal("No error")
	}
	fmt.Println(err)
}

func TestMarshalMapBadKeyType(t *testing.T) {
	_, err := bencode.Marshal(map[int]int{0: 0})
	if err == nil {
		t.Fatal("No error")
	}
	fmt.Println(err)
}

func TestMarshalBadMap(t *testing.T) {
	_, err := bencode.Marshal(map[string]complex128{"complex": 0})
	if err == nil {
		t.Fatal("No error")
	}
	fmt.Println(err)
}

var testStruct TestStruct

func init() {
	testStruct = TestStruct{
		People:      []Person{{}, {}, {}},
		Thing:       []byte("Some bytes"),
		DoubleSlice: [][]int{{}, {}, {}},
		Time:        Time{time.Now()},
	}
	testStruct.Len = uint(len(testStruct.Thing))
}
