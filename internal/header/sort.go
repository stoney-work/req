package header

import (
	"net/textproto"
	"sort"
	"sync"
)

type KeyValues struct {
	Key    string
	Values []string
}

type sorter struct {
	order map[string]int
	kvs   []KeyValues
}

func (s *sorter) Len() int      { return len(s.kvs) }
func (s *sorter) Swap(i, j int) { s.kvs[i], s.kvs[j] = s.kvs[j], s.kvs[i] }
func (s *sorter) Less(i, j int) bool {
	if index, ok := s.order[textproto.CanonicalMIMEHeaderKey(s.kvs[i].Key)]; ok {
		i = index
	}
	if index, ok := s.order[textproto.CanonicalMIMEHeaderKey(s.kvs[j].Key)]; ok {
		j = index
	}
	return i < j
}

// kvsPool 复用 []KeyValues，避免每次 writeRequest 都重新分配。
// 初始容量 24 略大于 Chrome header 数量（~17），确保首次 append 不扩容。
var kvsPool = sync.Pool{
	New: func() any {
		s := make([]KeyValues, 0, 24)
		return &s
	},
}

// BuildOrderMap 根据有序 key 列表预计算 order map，供 SortKeyValuesCached 使用。
// 应在 client 初始化时调用一次，结果缓存在 Transport 中。
func BuildOrderMap(orderedKeys []string) map[string]int {
	order := make(map[string]int, len(orderedKeys))
	for i, key := range orderedKeys {
		order[textproto.CanonicalMIMEHeaderKey(key)] = i
	}
	return order
}

// SortKeyValues 原有接口保持不变，供未缓存场景兜底使用。
func SortKeyValues(kvs []KeyValues, orderedKeys []string) {
	order := BuildOrderMap(orderedKeys)
	SortKeyValuesCached(kvs, order)
}

// SortKeyValuesCached 使用预计算好的 order map 排序，避免每次请求重新 make(map)。
func SortKeyValuesCached(kvs []KeyValues, order map[string]int) {
	s := &sorter{
		order: order,
		kvs:   kvs,
	}
	sort.Sort(s)
}

// GetKvsFromPool 从 pool 取一个 []KeyValues 指针。
func GetKvsFromPool() *[]KeyValues {
	return kvsPool.Get().(*[]KeyValues)
}

// PutKvsToPool 将 []KeyValues 清空后归还 pool，保留底层数组内存。
func PutKvsToPool(p *[]KeyValues) {
	*p = (*p)[:0]
	kvsPool.Put(p)
}
