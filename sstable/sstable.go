package sstable

import (
	"io"
	"os"

	"github.com/merlin82/leveldb/internal"
	"github.com/merlin82/leveldb/sstable/block"
)

type SsTable struct {
	index  *block.Block
	footer Footer
	file   *os.File
}

func Open(fileName string) (*SsTable, error) {
	var table SsTable
	var err error
	// sstbale文件描述符
	table.file, err = os.Open(fileName)
	if err != nil {
		return nil, err
	}
	// 文件大小
	stat, _ := table.file.Stat()
	// Read the footer block
	footerSize := int64(table.footer.Size())
	if stat.Size() < footerSize {
		return nil, internal.ErrTableFileTooShort
	}
	// 最后24B字节数据
	_, err = table.file.Seek(-footerSize, io.SeekEnd)
	if err != nil {
		return nil, err
	}
	// 24B数据转为footer结构体数据
	err = table.footer.DecodeFrom(table.file)
	if err != nil {
		return nil, err
	}
	// footer里面有meta和index的offset和size数据
	// 现在只实现了index没有meta，否则也需要读取
	// 读取第一个block
	table.index = table.readBlock(table.footer.IndexHandle)
	return &table, nil
}

func (table *SsTable) NewIterator() *Iterator {
	var it Iterator
	it.table = table
	it.indexIter = table.index.NewIterator()
	return &it
}

func (table *SsTable) Get(key []byte) ([]byte, error) {
	it := table.NewIterator()
	it.Seek(key)
	if it.Valid() {
		internalKey := it.InternalKey()
		if internal.UserKeyComparator(key, internalKey.UserKey) == 0 {
			// 判断valueType
			if internalKey.Type == internal.TypeValue {
				return internalKey.UserValue, nil
			} else {
				return nil, internal.ErrDeletion
			}
		}
	}
	return nil, internal.ErrNotFound
}

func (table *SsTable) readBlock(blockHandle BlockHandle) *block.Block {
	p := make([]byte, blockHandle.Size)
	n, err := table.file.ReadAt(p, int64(blockHandle.Offset))
	if err != nil || uint32(n) != blockHandle.Size {
		return nil
	}

	return block.New(p)
}
