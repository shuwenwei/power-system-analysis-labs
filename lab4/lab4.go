package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
)

type Branch struct {
	// 节点1
	Node1 int `json:"node_1"`
	// 节点2
	Node2 int `json:"node_2"`
	// 电阻
	Resistance float64 `json:"resistance"`
	// 电抗
	Reactance float64 `json:"reactance"`
	// 导纳
	Admittance float64 `json:"admittance"`
}

type PowerNetwork struct {
	// 正序
	Grid1 []Branch `json:"grid1"`
	F1    int      `json:"f1"`
	// 负序
	Grid2 []Branch `json:"grid2"`
	F2    int      `json:"f2"`
	// 零序
	Grid0 []Branch `json:"grid0"`
	F0    int      `json:"f0"`
}

type Parser struct {
	SB       float64
	Vav      float64
	branches []Branch
	nodeNum  int
	resultY  [][]complex128
	resultZ  *ComplexMatrix
}

type ComplexMatrix struct {
	m [][]complex128
}

func NewComplexMatrix(row int, col int) *ComplexMatrix {
	cm := new(ComplexMatrix)
	cm.m = make([][]complex128, row)
	for i := 0; i < row; i++ {
		cm.m[i] = make([]complex128, col)
	}
	return cm
}

// 输入参数为行和列的设值方式
func (cm *ComplexMatrix) rcSet(row, column int, v complex128) {
	cm.m[row-1][column-1] = v
}

// 输入参数为行和列的取值方式
func (cm *ComplexMatrix) rcAt(row, column int) complex128 {
	return cm.m[row-1][column-1]
}

func (p *Parser) computeResultY() {
	for i := 0; i < len(p.branches); i++ {
		branch := p.branches[i]
		if branch.Admittance != 0 {
			p.resultY[branch.Node1-1][branch.Node1-1] += -complex(0, branch.Admittance)
			p.resultY[branch.Node2-1][branch.Node2-1] += -complex(0, branch.Admittance)
		}
		if node, isGroundBranch := p.isGroundBranch(branch); isGroundBranch {
			// 改变-yi0的值
			if branch.Resistance != 0 || branch.Reactance != 0 {
				p.resultY[node-1][node-1] += -1 / complex(branch.Resistance, branch.Reactance)
			}
		} else {
			// 计算Yij
			p.computeYij(branch)
		}
	}
	for i := 1; i <= p.nodeNum; i++ {
		p.computeYii(i)
	}
}

func (p *Parser) isGroundBranch(branch Branch) (int, bool) {
	if branch.Node1 == 0 {
		return branch.Node2, true
	} else if branch.Node2 == 0 {
		return branch.Node1, true
	}
	return 0, false
}

func (p *Parser) computeYij(branch Branch) {
	Yij := -1 / complex(branch.Resistance, branch.Reactance)
	p.resultY[branch.Node1-1][branch.Node2-1] += Yij
	p.resultY[branch.Node2-1][branch.Node1-1] += Yij
}

func (p *Parser) computeYii(node int) {
	Yii := complex(0, 0)
	// Yii = -(-yi0 + Yi1 + Yi2 + ...)
	for i := 0; i < p.nodeNum; i++ {
		Yii -= p.resultY[node-1][i]
	}
	p.resultY[node-1][node-1] = Yii
}

func (p *Parser) printNormalResultMatrix() {
	p.printResultMatrix(p.resultY)
}

func (p *Parser) printResultMatrix(result [][]complex128) {
	for i := 0; i < p.nodeNum; i++ {
		for j := 0; j < p.nodeNum; j++ {
			c := result[i][j]
			fmt.Printf("%.3f", real(c))
			if imag(c) >= 0 {
				fmt.Printf(" + ")
			} else {
				fmt.Printf(" - ")
			}
			fmt.Printf("j%.3f\t\t", math.Abs(imag(c)))
		}
		fmt.Println()
	}
}

func NewParser(branches []Branch) *Parser {
	p := &Parser{
		branches: branches,
	}
	for i := 0; i < len(p.branches); i++ {
		branch := p.branches[i]
		if branch.Node1 > p.nodeNum {
			p.nodeNum = branch.Node1
		}
		if branch.Node2 > p.nodeNum {
			p.nodeNum = branch.Node2
		}
	}
	for i := 0; i < p.nodeNum; i++ {
		p.resultY = append(p.resultY, make([]complex128, p.nodeNum))
	}
	return p
}

func (p *Parser) computeZ(l, d, u *ComplexMatrix) *ComplexMatrix {
	Z := NewComplexMatrix(p.nodeNum, p.nodeNum)
	for j := 1; j <= p.nodeNum; j++ {
		p.computeZj(j, l, d, u, Z)
	}
	return Z
}

func (p *Parser) computeZj(j int, l, d, u, Z *ComplexMatrix) {
	length := p.nodeNum
	f := NewComplexMatrix(1, length)
	h := NewComplexMatrix(1, length)
	for i := 1; i <= length; i++ {
		if i < j {
			f.rcSet(1, i, 0)
		} else if i == j {
			f.rcSet(1, i, 1)
		} else {
			sum := complex(0, 0)
			for k := j; k <= i-1; k++ {
				sum -= l.rcAt(i, k) * f.rcAt(1, k)
			}
			f.rcSet(1, i, sum)
		}
	}
	for i := 1; i <= length; i++ {
		if i < j {
			h.rcSet(1, i, 0)
		} else {
			h.rcSet(1, i, f.rcAt(1, i)/d.rcAt(i, i))
		}
	}
	for i := length; i >= 1; i-- {
		sumUikZkj := complex(0, 0)
		for k := i + 1; k <= length; k++ {
			sumUikZkj += u.rcAt(i, k) * Z.rcAt(k, j)
		}
		Z.rcSet(i, j, h.rcAt(1, i)-sumUikZkj)
	}
}

func (p *Parser) LDU() (l *ComplexMatrix, d *ComplexMatrix, u *ComplexMatrix) {
	L := NewComplexMatrix(p.nodeNum, p.nodeNum)
	D := NewComplexMatrix(p.nodeNum, p.nodeNum)
	U := NewComplexMatrix(p.nodeNum, p.nodeNum)
	// 计算Li1,并设置L和U对角线上值为1
	for i := 1; i <= p.nodeNum; i++ {
		L.rcSet(i, 1, p.resultY[i-1][0]/p.resultY[0][0])
		L.rcSet(i, i, 1)
		U.rcSet(i, i, 1)
	}
	for i := 1; i <= p.nodeNum; i++ {
		// 设置dii
		Uki2Dkk := complex(0, 0)
		for k := 1; k <= i-1; k++ {
			Uki2Dkk += U.rcAt(k, i) * U.rcAt(k, i) * D.rcAt(k, k)
		}
		aii := p.resultY[i-1][i-1]
		D.rcSet(i, i, aii-Uki2Dkk)

		// 设置uij,(i = 1, 2, ..., n-1    j = i + 1, ..., n)
		if i != p.nodeNum {
			for j := i + 1; j <= p.nodeNum; j++ {
				UkiUkjDkk := complex(0, 0)
				for k := 1; k <= i-1; k++ {
					UkiUkjDkk += U.rcAt(k, i) * U.rcAt(k, j) * D.rcAt(k, k)
				}
				aij := p.resultY[i-1][j-1]
				dii := D.rcAt(i, i)
				U.rcSet(i, j, (aij-UkiUkjDkk)/dii)
			}
		}

		// lij的计算从i=2开始
		if i == 1 {
			continue
		}
		// 设置lij(i = 2, 3, ..., n   j = 1, 2, ..., i-1)
		for j := 1; j <= i-1; j++ {
			LikLjkDkk := complex(0, 0)
			for k := 1; k <= j-1; k++ {
				LikLjkDkk += L.rcAt(i, k) * L.rcAt(j, k) * D.rcAt(k, k)
			}
			aij := p.resultY[i-1][j-1]
			djj := D.rcAt(j, j)
			L.rcSet(i, j, (aij-LikLjkDkk)/djj)
		}
	}
	return L, D, U
}

func (p *Parser) computeResult() {
	p.computeResultY()
	p.resultZ = p.computeZ(p.LDU())
}

func (p *Parser) computeShortIf(f int) complex128 {
	zf := complex(0, 0)
	return 1 / (p.resultZ.rcAt(f, f) + zf)
}

func (p *Parser) computeAllNodeShortU(f int) []complex128 {
	zf := complex(0, 0)
	Zff := p.resultZ.rcAt(f, f)
	U := make([]complex128, p.nodeNum)
	for i := 1; i <= len(U); i++ {
		U[i-1] = 1 - (p.resultZ.rcAt(i, f) / (Zff + zf))
	}
	return U
}

func (p *Parser) computeIij(U []complex128) map[string]complex128 {
	Iij := map[string]complex128{}
	for i := 1; i <= p.nodeNum; i++ {
		for j := 1; j <= p.nodeNum; j++ {
			if i == j {
				continue
			}
			smallNodeNum := int(math.Min(float64(i), float64(j)))
			largeNodeNum := int(math.Max(float64(i), float64(j)))
			name := fmt.Sprintf("I%d%d", smallNodeNum, largeNodeNum)
			if _, exist := Iij[name]; exist {
				continue
			}
			yij := p.resultY[i-1][j-1]
			if i < j {
				Iij[name] = (U[i-1] - U[j-1]) * yij
			} else {
				Iij[name] = (U[j-1] - U[i-1]) * yij
			}
		}
	}
	return Iij
}

func main() {
	fmt.Println("输入文件的路径:")
	var path string
	fmt.Scanln(&path)
	network := importPowerNetworkFromFile(path)
	parser1 := NewParser(network.Grid1)
	parser1.computeResult()
	parser2 := NewParser(network.Grid2)
	parser2.computeResult()
	parser0 := NewParser(network.Grid0)
	parser0.computeResult()
	Zff1 := parser1.resultZ.rcAt(network.F1, network.F1)
	Zff2 := parser2.resultZ.rcAt(network.F2, network.F2)
	Zff0 := parser0.resultZ.rcAt(network.F0, network.F0)
	fmt.Printf("Zff(1): %v\n", Zff1)
	fmt.Printf("Zff(2): %v\n", Zff2)
	fmt.Printf("Zff(0): %v\n", Zff0)
	Ifa1 := 1 / (Zff2 + Zff1 + Zff0)
	fmt.Printf("Ifa(1) = %v\n", Ifa1)
	If1 := 3 * Ifa1
	fmt.Printf("If(1) = %v\n", If1)
	fmt.Printf("Ifa = %v\n", If1)

	VG11 := parser1.computeAllNodeShortU(4)[0] - parser1.resultZ.rcAt(1, network.F1) * Ifa1
	VG12 := -parser2.resultZ.rcAt(1, network.F1) * Ifa1
	fmt.Printf("Vg1 = %v\n", VG11+VG12)
	VG21 := parser1.computeAllNodeShortU(4)[5-1] - parser1.resultZ.rcAt(5, network.F1) * Ifa1
	VG22 := parser2.resultZ.rcAt(5, network.F1) * Ifa1
	fmt.Printf("Vg2 = %v\n", VG21+ VG22)
	fmt.Printf("Vg2 = %v\n", VG2)
}

func importPowerNetworkFromFile(path string) PowerNetwork {
	file, err := os.Open(path)
	defer func() {
		if file != nil {
			file.Close()
		}
	}()
	if err != nil {
		log.Fatal("打开文件失败")
	}
	decoder := json.NewDecoder(file)
	var network PowerNetwork
	if err := decoder.Decode(&network); err != nil {
		log.Fatal("解析失败")
	}
	return network
}
