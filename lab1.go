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
	// 添加阻抗的支路
	p.branches = append(p.branches, branch)
	//// 添加电纳的支路
	//admittance := 0.5 * circuit.B * circuit.L * p.Vav * p.Vav / p.SB
	//admittanceBranch1 := Branch{Node1: circuit.Node1, Node2: 0, Admittance: admittance}
	//admittanceBranch2 := Branch{Node1: 0, Node2: circuit.Node2, Admittance: admittance}
	//p.branches = append(p.branches, admittanceBranch1, admittanceBranch2)
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
			fmt.Printf("node1: %v node2: %v\n", branch.Node1, branch.Node2)
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

func (p *Parser) printResultMatrix() {
	for i := 0; i < p.nodeNum; i++ {
		for j := 0; j < p.nodeNum; j++ {
			c := p.result[i][j]
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

func main() {
	network := importPowerNetworkFromFile("/home/sww/GolandProjects/power-system-analysis-labs/test2.json")
	parser := NewParser(network)
	parser.computeResult()
	parser.printResultMatrix()
	parser.PrintShortCircuit(3)
}

func importPowerNetworkFromFile(path string) PowerNetwork {
	file, err := os.Open(path)
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

func importBranchesFromFile(path string) []Branch {
	file, err := os.Open(path)
	if err != nil {
		log.Fatal("打开文件失败")
	}
	decoder := json.NewDecoder(file)
	var branches []Branch
	if err := decoder.Decode(&branches); err != nil {
		log.Fatal("解码失败")
	}
	return branches
}
