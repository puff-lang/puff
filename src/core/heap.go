package core

type Map struct {
	object int
	addr   Addr
}

type hItem struct {
	NumObj  Object // Numbers of objects in Heap
	lUnAddr []Addr // List of unused Addresses
	listMap []Map  // an association list mapping addresses to objects.
}

type Heap struct {
	hItems [10]hItem
	index  int
}

// func HInitial() Heap{
// 	var h Heap
// 	h.index = -1
// 	return h
// }

// func (h *Heap) HAlloc(item hItem) {
// 	h.index = h.index + 1
// 	h.hItems[h.index].NumObj = item.NumObj;
// 	h.hItems[h.index].lUnAddr = item.lUnAddr;
// 	h.hItems[h.index].listMap = item.listMap;
// }

// func (h *Heap) HLookup(a Addr) bool {
// 	for i := 0; i <= h.index; i++ {
// 		if h.hItems[i].listMap[0].addr == a {
// 			return true
// 		}
// 	}
// 	return false
// }

// func (h *Heap) HFree(i int) {
// 	tmphItem := h.hItems[h.index]
// 	h.hItems[h.index] = h.hItems[i]
// 	h.hItems[i] = tmphItem
// 	h.index = h.index-1
// 	fmt.Println("xyz: ",i, h.index)
// }
