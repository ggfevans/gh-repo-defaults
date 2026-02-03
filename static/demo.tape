# gh-mint demo
# Run: vhs static/demo.tape

Output static/hero.gif

Set Shell bash
Set FontSize 16
Set Width 900
Set Height 550
Set Padding 20
Set Theme "Catppuccin Mocha"
Set TypingSpeed 40ms
Set CursorBlink false

# Show what this thing does
Type "gh-mint --help"
Enter
Sleep 2s

# List available profiles
Type "gh-mint profiles list"
Enter
Sleep 2.5s

# Inspect a profile
Type "gh-mint profiles show oss"
Enter
Sleep 3s

# Show the create command
Type "gh-mint create --help"
Enter
Sleep 2s

# One-liner example
Type "# One command. Fully configured repo."
Enter
Sleep 500ms
Type "gh-mint create my-project --profile oss --public"
Sleep 2s
