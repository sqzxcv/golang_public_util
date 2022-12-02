package Slice

//InsertElementToSlice 在切片中的指定位置插入指定元素
func InsertElementToSlice(slice []interface{}, value interface{}, place int) []interface{} {
    if place < 0 {
        place = 0
    }
    if place < len(slice) {
        a := append(slice, nil)
        copy(a[place+1:], a[place:])
        a[place] = value
        return a
    } else {
        return append(slice, value)
    }
}
