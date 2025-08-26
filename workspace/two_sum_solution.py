"""
Two Sum Problem Solution

Given an array of integers nums and an integer target, 
return indices of the two numbers such that they add up to target.

You may assume that each input would have exactly one solution, 
and you may not use the same element twice.

You can return the answer in any order.
"""

def two_sum_brute_force(nums, target):
    """
    Brute force solution - O(n^2) time complexity
    """
    for i in range(len(nums)):
        for j in range(i + 1, len(nums)):
            if nums[i] + nums[j] == target:
                return [i, j]
    return []

def two_sum_hash_map(nums, target):
    """
    Optimal solution using hash map - O(n) time complexity
    """
    num_map = {}
    for i, num in enumerate(nums):
        complement = target - num
        if complement in num_map:
            return [num_map[complement], i]
        num_map[num] = i
    return []

def two_sum_two_pointers(nums, target):
    """
    Two pointers solution (requires sorted array) - O(n log n) time complexity
    """
    # Create a list of (number, original_index) pairs
    indexed_nums = [(num, idx) for idx, num in enumerate(nums)]
    # Sort by number
    indexed_nums.sort(key=lambda x: x[0])
    
    left, right = 0, len(indexed_nums) - 1
    
    while left < right:
        current_sum = indexed_nums[left][0] + indexed_nums[right][0]
        if current_sum == target:
            return [indexed_nums[left][1], indexed_nums[right][1]]
        elif current_sum < target:
            left += 1
        else:
            right -= 1
    return []

# Example usage and test cases
if __name__ == "__main__":
    # Test cases
    test_cases = [
        ([2, 7, 11, 15], 9),  # Expected: [0, 1]
        ([3, 2, 4], 6),       # Expected: [1, 2]
        ([3, 3], 6),          # Expected: [0, 1]
        ([1, 2, 3, 4], 7),    # Expected: [2, 3]
    ]
    
    print("Two Sum Problem Solutions")
    print("=" * 50)
    
    for nums, target in test_cases:
        print(f"\nInput: nums = {nums}, target = {target}")
        
        # Brute force solution
        result_bf = two_sum_brute_force(nums, target)
        print(f"Brute Force: {result_bf}")
        
        # Hash map solution
        result_hm = two_sum_hash_map(nums, target)
        print(f"Hash Map: {result_hm}")
        
        # Two pointers solution (note: requires sorting)
        result_tp = two_sum_two_pointers(nums, target)
        print(f"Two Pointers: {result_tp}")
        
        print("-" * 30)

# Additional explanation
"""
Time Complexity Analysis:
1. Brute Force: O(n^2) - Nested loops
2. Hash Map: O(n) - Single pass with constant time lookups
3. Two Pointers: O(n log n) - Sorting + linear scan

Space Complexity Analysis:
1. Brute Force: O(1) - No extra space
2. Hash Map: O(n) - Hash map storage
3. Two Pointers: O(n) - Storage for indexed array

Recommendation:
- For most cases, use the hash map solution as it provides the best time complexity
- Two pointers is useful when the array is already sorted or when you need to find multiple pairs
- Brute force should only be used for small input sizes
"""