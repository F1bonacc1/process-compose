package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/alecthomas/chroma/v2/quick"
	"github.com/f1bonacc1/process-compose/src/recipe"
	"github.com/spf13/cobra"
)

var (
	recipeManager       *recipe.Manager
	forceFlag           bool
	outPath             string
	tagsFlag            []string
	authorFlag          string
	syntaxHighlightFlag bool
)

// recipeCmd represents the recipe command
var recipeCmd = &cobra.Command{
	Use:   "recipe",
	Short: "Manage process-compose recipes",
	Long: `Manage process-compose recipes from the community repository.

Recipes are pre-configured process-compose.yaml files for common use cases
like databases, message queues, and other services.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		recipeManager = recipe.NewManager("")
		if err := recipeManager.Initialize(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize recipe manager: %v\n", err)
			os.Exit(1)
		}
	},
}

// pullRecipeCmd represents the pull command
var pullRecipeCmd = &cobra.Command{
	Use:   "pull [recipe-name]",
	Short: "Pull a recipe from the repository",
	Long: `Download and install a recipe from the process-compose recipes repository.

The recipe will be downloaded to your local recipes directory and can be used
with 'process-compose -f ~/.process-compose/recipes/[recipe-name]/process-compose.yaml'`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		recipeName := args[0]
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		fmt.Printf("Pulling recipe '%s'...\n", recipeName)

		if err := recipeManager.PullRecipe(ctx, recipeName, forceFlag, outPath); err != nil {
			return fmt.Errorf("failed to pull recipe: %w", err)
		}

		return nil
	},
}

// searchRecipeCmd represents the search command
var searchRecipeCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search for recipes in the repository",
	Long: `Search for recipes in the process-compose recipes repository.

You can search by name, description, author, or tags.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		filter := recipe.SearchFilter{
			Tags:   tagsFlag,
			Author: authorFlag,
		}

		if len(args) > 0 {
			query := args[0]
			filter.Name = query
			filter.Description = query
		}

		fmt.Println("Searching recipes...")

		recipes, err := recipeManager.SearchRecipes(ctx, filter)
		if err != nil {
			return fmt.Errorf("failed to search recipes: %w", err)
		}

		if len(recipes) == 0 {
			fmt.Println("No recipes found matching your criteria.")
			return nil
		}

		// Display results in a table
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tDESCRIPTION\tVERSION\tAUTHOR\tTAGS")
		fmt.Fprintln(w, "----\t-----------\t-------\t------\t----")

		for _, recipe := range recipes {
			tags := strings.Join(recipe.Tags, ", ")
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				recipe.Name,
				recipe.Description,
				recipe.Version,
				recipe.Author,
				tags)
		}

		w.Flush()
		return nil
	},
}

// listCmd represents the list command
var listRecipesCmd = &cobra.Command{
	Use:   "list",
	Short: "List locally installed recipes",
	Long:  `List all recipes that have been pulled and are available locally.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		recipes, err := recipeManager.ListLocalRecipes()
		if err != nil {
			return fmt.Errorf("failed to list local recipes: %w", err)
		}

		if len(recipes) == 0 {
			fmt.Println("No recipes installed locally.")
			fmt.Println("Use 'process-compose recipe search' to find recipes, then 'process-compose recipe pull <name>' to install them.")
			return nil
		}

		// Display results in a table
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tDESCRIPTION\tVERSION\tPATH")
		fmt.Fprintln(w, "----\t-----------\t-------	----")

		for _, recipe := range recipes {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				recipe.Name,
				recipe.Description,
				recipe.Version,
				recipe.Path)
		}

		w.Flush()
		return nil
	},
}

// showRecipeCmd represents the show command
var showRecipeCmd = &cobra.Command{
	Use:   "show [recipe-name]",
	Short: "Show the content of a local recipe",
	Long:  `Display the process-compose.yaml content of a locally installed recipe to stdout.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		recipeName := args[0]

		content, err := recipeManager.GetRecipeContent(recipeName)
		if err != nil {
			return fmt.Errorf("failed to get recipe content: %w", err)
		}

		if syntaxHighlightFlag {
			return quick.Highlight(os.Stdout, string(content), "yaml", "terminal", "monokai")
		}

		fmt.Print(string(content))
		return nil
	},
}

// removeRecipeCmd represents the remove command
var removeRecipeCmd = &cobra.Command{
	Use:     "remove [recipe-name]",
	Aliases: []string{"rm", "delete"},
	Short:   "Remove a locally installed recipe",
	Long:    `Remove a recipe from your local recipes directory.`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		recipeName := args[0]

		if err := recipeManager.RemoveRecipe(recipeName); err != nil {
			return fmt.Errorf("failed to remove recipe: %w", err)
		}

		fmt.Printf("Recipe '%s' removed successfully.\n", recipeName)
		return nil
	},
}

func init() {
	// Add recipe command to root
	rootCmd.AddCommand(recipeCmd)

	// Add subcommands
	recipeCmd.AddCommand(pullRecipeCmd)
	recipeCmd.AddCommand(searchRecipeCmd)
	recipeCmd.AddCommand(listRecipesCmd)
	recipeCmd.AddCommand(showRecipeCmd)
	recipeCmd.AddCommand(removeRecipeCmd)

	// Add flags
	pullRecipeCmd.Flags().BoolVarP(&forceFlag, "force", "f", false, "Force pull even if recipe exists locally")
	pullRecipeCmd.Flags().StringVarP(&outPath, "output", "o", "", "Output path for the recipe")

	searchRecipeCmd.Flags().StringSliceVarP(&tagsFlag, "tags", "t", nil, "Filter by tags")
	searchRecipeCmd.Flags().StringVarP(&authorFlag, "author", "a", "", "Filter by author")

	showRecipeCmd.Flags().BoolVarP(&syntaxHighlightFlag, "syntax-highlight", "s", false, "Highlight the recipe yaml syntax")
}
