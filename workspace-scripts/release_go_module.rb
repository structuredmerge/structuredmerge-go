#!/usr/bin/env ruby
# frozen_string_literal: true

require "open3"
require "optparse"
require "fileutils"
require "shellwords"

GO_REPO = File.expand_path("..", __dir__)
STRUCTUREDMERGE_ROOT = File.expand_path("..", GO_REPO)
GO_MODULE = "github.com/structuredmerge/structuredmerge-go"

PACKAGES = [
  "treehaver",
  "astmerge",
  "plainmerge",
  "jsonmerge",
  "yamlmerge",
  "tomlmerge",
  "markdownmerge",
  "rubymerge",
  "gomerge",
  "rustmerge",
  "typescriptmerge",
  "asttemplate",
  "binarymerge",
  "zipmerge",
  "goccygoyamlmerge",
  "pigeontomlmerge",
  "goldmarkmerge",
  "goparsermerge",
  "kettlegomodder",
].freeze

options = {
  push: true,
  push_git: false,
  skip_tests: false,
  tag: false,
}

parser = OptionParser.new do |opts|
  opts.banner = "Usage: release_go_module.rb --version VERSION [options]"

  opts.on("--version VERSION", "Go module version to release, for example 0.1.0 or v0.1.0") do |version|
    options[:version] = version
  end

  opts.on("--push", "Create/push the module tag after local validation (default)") do
    options[:push] = true
    options[:tag] = true
  end

  opts.on("--no-push", "--dry-run", "Validate locally without creating or pushing a tag") do
    options[:push] = false
  end

  opts.on("--push-git", "Push the Go repo branch before pushing the release tag") do
    options[:push_git] = true
  end

  opts.on("--tag", "Create the vVERSION tag locally") do
    options[:tag] = true
  end

  opts.on("--skip-tests", "Skip go test checks") do
    options[:skip_tests] = true
  end

  opts.on("--only PACKAGE", "Validate only one package directory from the publish order") do |package_name|
    options[:only] = package_name
  end

  opts.on("--start-at PACKAGE", "Start validation at a package directory in the publish order") do |package_name|
    options[:start_at] = package_name
  end
end

parser.parse!

def sh(cmd)
  Shellwords.join(cmd)
end

def run!(cmd, chdir:, env: {})
  puts "\n$ #{sh(cmd)}"
  return if system(env, *cmd, chdir: chdir)

  raise "Command failed: #{sh(cmd)}"
end

def capture!(cmd, chdir:)
  output, status = Open3.capture2e(*cmd, chdir: chdir)
  raise "Command failed: #{sh(cmd)}\n#{output}" unless status.success?

  output
end

def ensure_clean_git!
  status = capture!(%w[git status --porcelain], chdir: GO_REPO)
  return if status.empty?

  raise "Go repo has uncommitted changes; commit before releasing.\n#{status}"
end

def normalize_version(version)
  raise "--version is required for Go module releases" unless version

  bare = version.delete_prefix("v")
  raise "Version must look like MAJOR.MINOR.PATCH, got #{version.inspect}" unless bare.match?(/\A\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?\z/)

  "v#{bare}"
end

def version_minor(tag_name)
  bare = tag_name.delete_prefix("v")
  match = bare.match(/\A(\d+)\.(\d+)\.\d+(?:[-+].*)?\z/)
  raise "Go module version #{tag_name.inspect} is not a simple semver version" unless match

  "#{match[1]}.#{match[2]}"
end

def ensure_one_minor_version!(package_versions)
  minor_versions = package_versions.group_by { |_package_name, tag_name| version_minor(tag_name) }
  return if minor_versions.one?

  details = minor_versions.map do |minor, grouped_versions|
    "#{minor}: #{grouped_versions.map { |package_name, tag_name| "#{package_name}@#{tag_name}" }.join(", ")}"
  end
  raise "Go packages must share one major.minor module version before release.\n#{details.join("\n")}"
end

def tag_exists?(tag_name)
  local = system("git", "rev-parse", "-q", "--verify", "refs/tags/#{tag_name}",
    chdir: GO_REPO, out: File::NULL, err: File::NULL)
  remote = system("git", "ls-remote", "--exit-code", "--tags", "origin", "refs/tags/#{tag_name}",
    chdir: GO_REPO, out: File::NULL, err: File::NULL)
  local || remote
end

def go_proxy_version_exists?(tag_name)
  output, status = Open3.capture2e("go", "list", "-m", "#{GO_MODULE}@#{tag_name}", chdir: GO_REPO)
  status.success? && output.include?(tag_name)
end

raise "Could not find Go repo at #{GO_REPO}" unless Dir.exist?(GO_REPO)

tag_name = normalize_version(options[:version])
ensure_one_minor_version!(PACKAGES.to_h { |package_name| [package_name, tag_name] })

selected_packages = PACKAGES.dup
if options[:only]
  selected_packages.select! { |package_name| package_name == options[:only] }
  raise "Unknown package for --only: #{options[:only]}" if selected_packages.empty?
end

if options[:start_at]
  start_index = selected_packages.index(options[:start_at])
  raise "Unknown package for --start-at: #{options[:start_at]}" unless start_index

  selected_packages = selected_packages.drop(start_index)
end

puts "Go module: #{GO_MODULE}"
puts "Go module version: #{tag_name}"
puts "Selected packages: #{selected_packages.join(", ")}"
puts "Tag push: #{options[:push] ? "enabled; pass --no-push for local validation only" : "disabled; local validation only"}"

ensure_clean_git! if options[:push] || options[:push_git] || options[:tag]

selected_packages.each do |package_name|
  raise "Missing Go package directory: #{package_name}" unless Dir.exist?(File.join(GO_REPO, package_name))
end

unless options[:skip_tests]
  selected_packages.each do |package_name|
    run!(["go", "test", "./#{package_name}", "-run", "^$"], chdir: GO_REPO)
  end
  run!(["go", "test", "./...", "-run", "^$"], chdir: GO_REPO) if selected_packages == PACKAGES

  # The published language-pack Go module ships its native library; no sibling checkout is needed.
  selected_packages.each do |package_name|
    run!(["go", "test", "-tags", "tspack", "./#{package_name}"], chdir: GO_REPO)
  end
  run!(["go", "test", "-tags", "tspack", "./..."], chdir: GO_REPO) if selected_packages == PACKAGES
end

if tag_exists?(tag_name)
  puts "\nSkipping tag creation; #{tag_name} already exists locally or on origin."
  if options[:push] && go_proxy_version_exists?(tag_name)
    puts "Go proxy already resolves #{GO_MODULE}@#{tag_name}."
  end
  exit 0
end

if options[:tag] || options[:push]
  run!(["git", "tag", "-a", tag_name, "-m", "Release Go module #{tag_name}"], chdir: GO_REPO)
else
  puts "\nLocal validation complete; no Go module tag was created."
  exit 0
end

if options[:push_git]
  run!(%w[git push origin HEAD], chdir: GO_REPO)
end

if options[:push]
  run!(["git", "push", "origin", tag_name], chdir: GO_REPO)
  puts "\nPushed Go module tag #{tag_name}. The public Go proxy may take a few minutes to observe it."
else
  puts "\nCreated local Go module tag #{tag_name}; pass --push to push it."
end
