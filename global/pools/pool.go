/**
 * @Author: jianxiaoli
 * @Description: goroutine池
 * @Date: 2018-02-02
 */

package pools

import (
	"github.com/panjf2000/ants/v2"
	"go-webshell/global/log"
)


var Pool *ants.Pool

func InitPool(poolNum int){
	var err error
	Pool,err = ants.NewPool(poolNum)
	if err != nil{
		log.Error("Init pool error by",err)
		panic(err)
	}
}
