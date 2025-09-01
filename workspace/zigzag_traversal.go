package main

import (
	"fmt"
)

// TreeNode represents a node in a binary tree
type TreeNode struct {
	Val   int
	Left  *TreeNode
	Right *TreeNode
}

// zigzagLevelOrder performs zigzag level order traversal of a binary tree
func zigzagLevelOrder(root *TreeNode) [][]int {
	if root == nil {
		return [][]int{}
	}

	var result [][]int
	queue := []*TreeNode{root}
	leftToRight := true

	for len(queue) > 0 {
		levelSize := len(queue)
		levelValues := make([]int, levelSize)

		for i := 0; i < levelSize; i++ {
			node := queue[0]
			queue = queue[1:]

			// Determine the index based on traversal direction
			index := i
			if !leftToRight {
				index = levelSize - 1 - i
			}
			levelValues[index] = node.Val

			// Add child nodes to queue
			if node.Left != nil {
				queue = append(queue, node.Left)
			}
			if node.Right != nil {
				queue = append(queue, node.Right)
			}
		}

		result = append(result, levelValues)
		leftToRight = !leftToRight // Reverse direction for next level
	}

	return result
}

// Helper function to create a binary tree for testing
func createTestTree() *TreeNode {
	/*
		Constructs the following tree:
		    3
		   / \
		  9   20
		     /  \
		    15   7
	*/
	return &TreeNode{
		Val: 3,
		Left: &TreeNode{
			Val: 9,
		},
		Right: &TreeNode{
			Val: 20,
			Left: &TreeNode{
				Val: 15,
			},
			Right: &TreeNode{
				Val: 7,
			},
		},
	}
}

func main() {
	// Create test tree
	tree := createTestTree()

	// Perform zigzag level order traversal
	result := zigzagLevelOrder(tree)

	// Print the result
	fmt.Println("Zigzag Level Order Traversal:")
	for i, level := range result {
		fmt.Printf("Level %d: %v\n", i, level)
	}

	// Expected output:
	// Level 0: [3]
	// Level 1: [20, 9]
	// Level 2: [15, 7]
}
