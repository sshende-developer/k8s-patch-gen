package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// Struct to represent the YAML file structure
type ResourceModifier struct {
	Version               string                 `yaml:"version"`
	ResourceModifierRules []ResourceModifierRule `yaml:"resourceModifierRules"`
}

type ResourceModifierRule struct {
	Conditions Conditions `yaml:"conditions"`
	Patches    []Patch    `yaml:"patches,omitempty"`
}

type Conditions struct {
	GroupResource     string            `yaml:"groupResource"`
	ResourceNameRegex string            `yaml:"resourceNameRegex,omitempty"`
	Namespaces        []string          `yaml:"namespaces,omitempty"`
	LabelSelector     map[string]string `yaml:"labelSelector,omitempty"`
}

type Patch struct {
	Operation string `yaml:"operation"`
	Path      string `yaml:"path"`
	From      string `yaml:"from,omitempty"` // Required for "copy" and "move"
	Value     string `yaml:"value,omitempty"`
}

// CLI command
var GenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a YAML file for Velero resource modifiers",
	Run:   generateYAML,
}

func generateYAML(cmd *cobra.Command, args []string) {
	var filename string
	var manifestFile string
	var manifestData map[string]interface{}

	// Ask if the user wants to refer to a manifest file
	var useManifest bool
	survey.AskOne(&survey.Confirm{Message: "📄 Do you want to refer to an existing manifest file?", Default: false}, &useManifest)

	if useManifest {
		survey.AskOne(&survey.Input{Message: "📄 Enter the path to the manifest file:"}, &manifestFile)
		fileContent, err := os.ReadFile(manifestFile)
		if err != nil {
			fmt.Println("❌ Error reading manifest file:", err)
			return
		}
		err = yaml.Unmarshal(fileContent, &manifestData)
		if err != nil {
			fmt.Println("❌ Error parsing manifest file:", err)
			return
		}
	}

	// Ask for the filename
	survey.AskOne(&survey.Input{Message: "📄 Enter filename to save the YAML (Default: patch.yaml):", Default: "patch.yaml"}, &filename)

	var resourceModifierRules []ResourceModifierRule

	for {
		var groupResource, resourceNameOption, namespaces, labelSelectorStr string

		// Auto-detect groupResource if manifest is provided
		if useManifest {
			detectedGroupResource := detectGroupResource(manifestData)
			survey.AskOne(&survey.Input{Message: fmt.Sprintf("🛠️ Detected GroupResource: %s (Press Enter to confirm or modify)", detectedGroupResource)}, &groupResource)
			if groupResource == "" {
				groupResource = detectedGroupResource
			}
		} else {
			survey.AskOne(&survey.Input{Message: "🛠️ Enter GroupResource (e.g., persistentvolumeclaims, *.apps, *.* for all. Default *.*):", Default: "*.*"}, &groupResource, survey.WithValidator(survey.Required))
		}

		// Ask for resource name specification method
		survey.AskOne(&survey.Select{
			Message: "🔍 How do you want to specify resource names?",
			Options: []string{"Starts with...", "Ends with...", "Contains...", "Exact match", "Provide custom regex", "No specific resource name"},
			Default: "No specific resource name",
		}, &resourceNameOption)

		// Handle resource name regex generation
		var resourceNameRegex string
		if resourceNameOption != "No specific resource name" {
			var resourceName string
			survey.AskOne(&survey.Input{Message: "🔠 Enter the resource name pattern:"}, &resourceName)

			if resourceNameOption == "Provide custom regex" {
				resourceNameRegex = resourceName
			} else {
				switch resourceNameOption {
				case "Starts with...":
					resourceNameRegex = fmt.Sprintf("^%s.*$", regexp.QuoteMeta(resourceName))
				case "Ends with...":
					resourceNameRegex = fmt.Sprintf("^.*%s$", regexp.QuoteMeta(resourceName))
				case "Contains...":
					resourceNameRegex = fmt.Sprintf(".*%s.*", regexp.QuoteMeta(resourceName))
				case "Exact match":
					resourceNameRegex = fmt.Sprintf("^%s$", regexp.QuoteMeta(resourceName))
				}
			}
		}

		// Handle namespaces (allow '*' for all namespaces)
		var namespaceList []string
		survey.AskOne(&survey.Input{
			Message: "📂 Enter namespaces (comma-separated, or '*' for all namespaces(Default is *)):",
			Default: "*",
		}, &namespaces)

		if namespaces == "*" {
			namespaceList = nil // If nil, it applies to all namespaces
		} else {
			namespaceList = strings.Split(namespaces, ",")
		}

		// Parse label selector into map
		labelSelector := make(map[string]string)
		if labelSelectorStr != "" {
			parts := strings.Split(labelSelectorStr, "=")
			if len(parts) == 2 {
				labelSelector[parts[0]] = parts[1]
			}
		}

		// Create a new resource rule
		rule := ResourceModifierRule{
			Conditions: Conditions{
				GroupResource:     groupResource,
				ResourceNameRegex: resourceNameRegex,
				Namespaces:        namespaceList,
				LabelSelector:     labelSelector,
			},
		}

		// Ask if the user wants to add patches
		for {
			var operation, path, value, from string

			// Allow user to view the manifest file just before asking for path
			if useManifest {
				var showFile bool
				survey.AskOne(&survey.Confirm{Message: "📜 Do you want to see the manifest file with line numbers before specifying the path?", Default: false}, &showFile)
				if showFile {
					printManifestWithLineNumbers(manifestFile)
				}
			}

			// Ask for path
			survey.AskOne(&survey.Input{Message: "📌 Enter the JSON path to modify (e.g., /spec/storageClassName):"}, &path)

			// Let the user choose a patch operation
			survey.AskOne(&survey.Select{
				Message: "🛠️ Choose patch operation:",
				Options: []string{"add", "remove", "replace", "copy", "move", "test"},
				Default: "replace",
			}, &operation)

			// Handle input differently based on operation
			switch operation {
			case "remove":
				value = "" // Skip value input for "remove"
			case "copy", "move":
				survey.AskOne(&survey.Input{Message: "🔀 Enter the source path (from):"}, &from)
			case "add":
				var entryMode string
				survey.AskOne(&survey.Select{
					Message: "📌 How do you want to provide the JSON value?",
					Options: []string{"Enter raw JSON", "Build interactively"},
					Default: "Build interactively",
				}, &entryMode)

				if entryMode == "Enter raw JSON" {
					var raw string
					survey.AskOne(&survey.Multiline{
						Message: "✏️ Paste the JSON payload to insert (array or object):",
					}, &raw)

					// Validate it's valid JSON
					var js interface{}
					if err := json.Unmarshal([]byte(raw), &js); err != nil {
						fmt.Println("❌ Invalid JSON:", err)
						return
					}

					// Re-encode with escaping
					escaped, err := json.Marshal(js)
					if err != nil {
						fmt.Println("❌ Error formatting JSON:", err)
						return
					}

					value = string(escaped) // escaped and safe for YAML
				} else {
					value = buildNestedJSON(nil)
				}

			default:
				survey.AskOne(&survey.Input{Message: "✏️ Enter new value:"}, &value)
			}

			// Add patch to rule
			if rule.Patches == nil {
				rule.Patches = []Patch{}
			}
			rule.Patches = append(rule.Patches, Patch{
				Operation: operation,
				Path:      path,
				From:      from,
				Value:     value,
			})

			var addMorePatches bool
			survey.AskOne(&survey.Confirm{Message: "➕ Do you want to add another patch to this resource?", Default: false}, &addMorePatches)
			if !addMorePatches {
				break
			}
		}

		resourceModifierRules = append(resourceModifierRules, rule)

		var addMoreRules bool
		survey.AskOne(&survey.Confirm{Message: "➕ Do you want to add another resource condition?", Default: false}, &addMoreRules)
		if !addMoreRules {
			break
		}
	}

	// Create YAML struct
	rules := ResourceModifier{
		Version:               "v1",
		ResourceModifierRules: resourceModifierRules,
	}

	// Convert to YAML
	yamlData, err := yaml.Marshal(&rules)
	if err != nil {
		fmt.Println("❌ Error generating YAML:", err)
		return
	}

	// Write to file
	err = os.WriteFile(filename, yamlData, 0644)
	if err != nil {
		fmt.Println("❌ Error saving YAML file:", err)
		return
	}

	fmt.Println("✅ YAML file generated successfully:", filename)
}

// Function to detect GroupResource from manifest file
func detectGroupResource(manifestData map[string]interface{}) string {
	if apiVersion, exists := manifestData["apiVersion"].(string); exists {
		parts := strings.Split(apiVersion, "/")
		if len(parts) == 2 {
			return fmt.Sprintf("%s.%s", parts[1], parts[0]) // Format: resource.group
		}
	}
	if kind, exists := manifestData["kind"].(string); exists {
		return strings.ToLower(kind)
	}
	return "unknown"
}

// Function to extract JSON path from line number in the manifest file
func extractPathFromLine(filename string, lineNumber int) string {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("❌ Error opening manifest file:", err)
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	currentLine := 0
	var pathStack []string
	indentLevels := map[int]string{} // Track indentation levels and corresponding keys

	for scanner.Scan() {
		currentLine++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Determine indentation level
		indentation := len(line) - len(strings.TrimLeft(line, " "))

		if strings.Contains(trimmed, ":") { // It's a key
			parts := strings.SplitN(trimmed, ":", 2)
			key := strings.TrimSpace(parts[0])

			// Remove deeper indentation levels
			for level := range indentLevels {
				if level >= indentation {
					delete(indentLevels, level)
				}
			}

			// Store the key at this indentation level
			indentLevels[indentation] = key
		}

		// Stop processing once we reach the target line
		if currentLine == lineNumber {
			// Build full path from indentation levels
			for i := 0; i <= indentation; i++ {
				if key, exists := indentLevels[i]; exists {
					pathStack = append(pathStack, key)
				}
			}

			// Construct final path
			if len(pathStack) > 0 {
				return "/" + strings.Join(pathStack, "/")
			}

			fmt.Println("⚠ Could not determine JSON path from line number.")
			return ""
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("❌ Error reading manifest file:", err)
		return ""
	}

	fmt.Println("⚠ Could not determine JSON path from line number.")
	return ""
}

// Function to print manifest file with line numbers
func printManifestWithLineNumbers(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("❌ Error opening manifest file:", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 1

	fmt.Println("\n📜 Manifest File Preview (with Line Numbers): ")
	for scanner.Scan() {
		fmt.Printf("%d: %s\n", lineNumber, scanner.Text())
		lineNumber++
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("❌ Error reading manifest file:", err)
	}
}

func buildNestedJSON(mockAskOne func(survey.Prompt, interface{}, ...survey.AskOpt) error) string {
	jsonMap := make(map[string]interface{})

	askFn := survey.AskOne
	if mockAskOne != nil {
		askFn = mockAskOne // Use mock function in tests
	}

	for {
		var key, valueType string
		askFn(&survey.Input{Message: "🔑 Enter key name (or press Enter to finish):"}, &key)

		// Stop if the user presses Enter without entering a key
		if key == "" {
			break
		}

		askFn(&survey.Select{
			Message: fmt.Sprintf("📌 What type of value should '%s' have?", key),
			Options: []string{"String", "Number", "Boolean", "Object"},
			Default: "String",
		}, &valueType)

		switch valueType {
		case "String":
			var value string
			askFn(&survey.Input{Message: fmt.Sprintf("✏️ Enter string value for '%s':", key)}, &value)
			jsonMap[key] = value
		case "Number":
			var valueStr string
			askFn(&survey.Input{Message: fmt.Sprintf("🔢 Enter numeric value for '%s':", key)}, &valueStr)

			var value float64
			_, err := fmt.Sscanf(valueStr, "%f", &value)
			if err != nil {
				fmt.Println("❌ Invalid number format, defaulting to 0")
				value = 0
			}

			jsonMap[key] = value
		case "Boolean":
			var value bool
			askFn(&survey.Confirm{Message: fmt.Sprintf("✅ Should '%s' be true?", key)}, &value)
			jsonMap[key] = value
		case "Object":
			fmt.Println("📌 Creating a nested object for:", key)
			nestedJSON := buildNestedJSON(mockAskOne)

			var nestedMap map[string]interface{}
			err := json.Unmarshal([]byte(nestedJSON), &nestedMap)
			if err != nil {
				fmt.Println("❌ Error parsing nested JSON:", err)
				nestedMap = make(map[string]interface{}) // Fallback to empty map
			}
			jsonMap[key] = nestedMap // Store as a map instead of a string
		}
	}

	// Convert the map to a JSON string
	jsonBytes, err := json.Marshal(jsonMap)
	if err != nil {
		fmt.Println("❌ Error generating JSON:", err)
		return "{}" // Return empty JSON if an error occurs
	}

	return string(jsonBytes)
}
