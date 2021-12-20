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
	// 如果Sn为0,则使用下面的参数计算
	Pn   float64 `json:"Pn"`
	Cos  float64 `json:"cos"`
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
	nodeNum  int
	result   [][]complex128
}

type PowerNetwork struct {
	SB              float64          `json:"SB"`
	Vav             float64          `json:"Vav"`
	PowerGenerators []PowerGenerator `json:"power_generators"`
	Circuits        []Circuit        `json:"circuits"`
	Transformers    []Transformer    `json:"transformers"`
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
	if generator.Sn == 0 {
		branch.Reactance = generator.Xd * p.SB / (generator.Pn / generator.Cos)
	} else {
		branch.Reactance = generator.Xd * p.SB / generator.Sn
	}
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
			p.result[branch.Node1-1][branch.Node1-1] += -complex(0, branch.Admittance)
			p.result[branch.Node2-1][branch.Node2-1] += -complex(0, branch.Admittance)
		}
		if node, isGroundBranch := p.isGroundBranch(branch); isGroundBranch {
			// 改变-yi0的值
			if branch.Resistance != 0 || branch.Reactance != 0 {
				p.result[node-1][node-1] += -1 / complex(branch.Resistance, branch.Reactance)
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
	p.result[branch.Node1-1][branch.Node2-1] = Yij
	p.result[branch.Node2-1][branch.Node1-1] = Yij
}

func (p *Parser) computeYii(node int) {
	Yii := complex(0, 0)
	// Yii = -(-yi0 + Yi1 + Yi2 + ...)
	for i := 0; i < p.nodeNum; i++ {
		Yii -= p.result[node-1][i]
	}
	p.result[node-1][node-1] = Yii
}

func (p *Parser) printNormalResultMatrix() {
	p.printResultMatrix(p.result)
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
		p.result = append(p.result, make([]complex128, p.nodeNum))
	}
	return p
}

func (p *Parser) PrintShortCircuit(node int) {
	for i := 0; i < p.nodeNum; i++ {
		for j := 0; j < p.nodeNum; j++ {
			// 跳过短路位置的行和列
			if i == node-1 || j == node-1 {
				continue
			}
			c := p.result[i][j]
			fmt.Printf("%.3f", real(c))
			if imag(c) >= 0 {
				fmt.Printf(" + ")
			} else {
				fmt.Printf(" - ")
			}
			fmt.Printf("j%.3f\t\t", math.Abs(imag(c)))
		}
		if i != node-1 {
			fmt.Println()
		}
	}
}

// 线路中点发生三相短路
func (p *Parser) printHalfShortCircuit(node1 int, node2 int) {
	var copyResult = make([][]complex128, p.nodeNum)
	for i := 0; i < len(p.result); i++ {
		copyRow := make([]complex128, p.nodeNum)
		copy(copyRow, p.result[i])
		copyResult[i] = copyRow
	}
	// 找到发生短路的branch
	var shortCircuit Circuit
	circuits := p.network.Circuits
	for i := 0; i < len(circuits); i++ {
		circuit := circuits[i]
		if (circuit.Node1 == node1 && circuit.Node2 == node2) || (circuit.Node2 == node1 && circuit.Node1 == node2) {
			shortCircuit = circuit
			break
		}
	}
	// Yii' = Yii - Yij - j0.25B
	copyResult[node1-1][node1-1] = copyResult[node1-1][node1-1] - copyResult[node1-1][node2-1] - complex(0, 0.25*shortCircuit.B)
	copyResult[node2-1][node2-1] = copyResult[node2-1][node2-1] - copyResult[node1-1][node2-1] - complex(0, 0.25*shortCircuit.B)
	// Yij' = 0
	copyResult[node1-1][node2-1] = 0
	copyResult[node2-1][node1-1] = 0
	p.printResultMatrix(copyResult)
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
	fmt.Println("输入发生三相短路的节点: ")
	var node int
	fmt.Scanln(&node)
	fmt.Printf("节点%d发生三相短路的节点导纳矩阵：\n", node)
	parser.PrintShortCircuit(node)
	var i int
	var j int
	fmt.Println("输入中点发生三相短路的两个节点的第一个")
	fmt.Scanln(&i)
	fmt.Println("输入中点发生三相短路的两个节点的第二个")
	fmt.Scanln(&j)
	fmt.Printf("线路%d-%d中点发生三相短路的节点导纳矩阵: \n", i, j)
	parser.printHalfShortCircuit(i, j)

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
