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

// 发电机
type PowerGenerator struct {
	Node int     `json:"node"`
	Sn   float64 `json:"Sn"`
	Xd   float64 `json:"xd"`
}

// 线路
type Circuit struct {
	Node1 int     `json:"node_1"`
	Node2 int     `json:"node_2"`
	R     float64 `json:"r"`
	X     float64 `json:"x"`
	B     float64 `json:"b"`
	L     float64 `json:"l"`
}

// 变压器
type Transformer struct {
	Node1 int     `json:"node_1"`
	Node2 int     `json:"node_2"`
	Sn    float64 `json:"Sn"`
	Vs    float64 `json:"Vs"`
}

type Parser struct {
	SB       float64
	Vav      float64
	network  PowerNetwork
	branches []Branch
	nodeNum int
	resultY [][]complex128
}

type PowerNetwork struct {
	SB              float64          `json:"SB"`
	Vav             float64          `json:"Vav"`
	PowerGenerators []PowerGenerator `json:"power_generators"`
	Circuits        []Circuit        `json:"circuits"`
	Transformers    []Transformer    `json:"transformers"`
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

func (p *Parser) parsePowerNetwork() {
	circuits := p.network.Circuits
	generators := p.network.PowerGenerators
	transformers := p.network.Transformers
	for i := 0; i < len(circuits); i++ {
		p.circuitArgsToBranch(circuits[i])
	}
	for i := 0; i < len(generators); i++ {
		p.powerGeneratorArgsToBranch(generators[i])
	}
	for i := 0; i < len(transformers); i++ {
		p.transformerArgsToBranch(transformers[i])
	}
}

func (p *Parser) circuitArgsToBranch(circuit Circuit) {
	branch := Branch{
		Node1: circuit.Node1,
		Node2: circuit.Node2,
	}
	// 计算电抗
	branch.Resistance = circuit.R * circuit.L * p.SB / (p.Vav * p.Vav)
	branch.Reactance = circuit.X * circuit.L * p.SB / (p.Vav * p.Vav)
	branch.Admittance = 0.5 * circuit.B * circuit.L * p.Vav * p.Vav / p.SB
	// 添加支路
	p.branches = append(p.branches, branch)
}

func (p *Parser) powerGeneratorArgsToBranch(generator PowerGenerator) {
	branch := Branch{
		Node1: generator.Node,
		Node2: 0,
	}
	branch.Reactance = generator.Xd * p.SB / generator.Sn
	p.branches = append(p.branches, branch)
}

func (p *Parser) transformerArgsToBranch(transformer Transformer) {
	branch := Branch{
		Node1: transformer.Node1,
		Node2: transformer.Node2,
	}
	branch.Reactance = (transformer.Vs / 100) * (p.SB / transformer.Sn)
	p.branches = append(p.branches, branch)
}

func (p *Parser) computeResult() {
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

func NewParser(network PowerNetwork) *Parser {
	p := &Parser{
		network: network,
	}
	p.SB = network.SB
	p.Vav = network.Vav
	p.parsePowerNetwork()
	for i := 0; i < len(p.branches); i++ {
		branch := p.branches[i]
		if branch.Node1 > p.nodeNum {
			p.nodeNum = branch.Node1
		}
		if branch.Node2 > p.nodeNum {
			p.nodeNum = branch.Node2
		}
	}
	fmt.Println(p.nodeNum)
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

func main() {
	fmt.Println("输入文件的路径:")
	var path string
	fmt.Scanln(&path)
	network := importPowerNetworkFromFile(path)
	parser := NewParser(network)
	parser.computeResult()
	fmt.Println("节点导纳矩阵：")
	parser.printNormalResultMatrix()
	fmt.Println("阻抗矩阵: ")
	Z := parser.computeZ(parser.LDU())
	parser.printResultMatrix(Z.m)
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
