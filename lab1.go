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

type Parser struct {
	branches []Branch
	nodeNum  int
	result   [][]complex128
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

func NewParser(path string) *Parser {
	p := &Parser{}
	p.branches = importBranchesFromFile(path)
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
	parser := NewParser("/home/sww/GolandProjects/power-system-analysis-labs/test.json")
	parser.computeResult()
	fmt.Println("节点导纳矩阵：")
	parser.printResultMatrix()
	fmt.Printf("节点%d发生三相短路的节点导纳矩阵\n", 3)
	parser.PrintShortCircuit(3)
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
