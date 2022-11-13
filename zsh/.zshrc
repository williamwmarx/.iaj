################################################################################
# Information
################################################################################
# Maintained by William W. Marx (oss@t3.atemp.studio)
# 🄯 Copyleft 2022, All Wrongs Reserved
# https://github.com/williamwmarx/.dotfiles


################################################################################
# Base Installs
################################################################################
# ----------------------- Oh-my-zsh syntax highlighting -----------------------
if [[ ! -d $HOME/.oh-my-zsh/custom/plugins/zsh-syntax-highlighting ]]; then
	git clone https://github.com/zsh-users/zsh-syntax-highlighting.git \
		${ZSH_CUSTOM:-~/.oh-my-zsh/custom}/plugins/zsh-syntax-highlighting
fi

# ------------------------- Oh-my-zsh autosuggestions -------------------------
if [[ ! -d $HOME/.oh-my-zsh/custom/plugins/zsh-autosuggestions ]]; then
	git clone https://github.com/zsh-users/zsh-autosuggestions \
		${ZSH_CUSTOM:-~/.oh-my-zsh/custom}/plugins/zsh-autosuggestions
fi


################################################################################
# Config
################################################################################
# ------------------------------ Oh-my-zsh Config -----------------------------
export ZSH=$HOME/.oh-my-zsh
ZSH_THEME="t3"
plugins=(zsh-syntax-highlighting zsh-autosuggestions)
source $ZSH/oh-my-zsh.sh

# ---------------------------- ARM64 Homebrew PATH ----------------------------
if [[ `uname -sp` == "Darwin arm" ]]; then export PATH="${PATH:+${PATH}:}/opt/homebrew/bin"; fi

# --------------------------------- Fzf Config --------------------------------
if [[ `which fzf &>/dev/null && echo "$?"` -eq 0 ]]; then
	if [[ ! "$PATH" == *$HOME/.fzf/bin* ]]; then export PATH="${PATH:+${PATH}:}$HOME/.fzf.bin"; fi

	[[ $- == *i* ]] && source "$HOME/.fzf/shell/completion.zsh" 2>/dev/null

	if [[ `which fd &>/dev/null && echo "$?"` -eq 0 ]]; then  # If fd is available, use it
		_fzf_compgen_path() { fd -HLE ".git" -E "Library" . $1 }
		_fzf_compgen_dir() { fd -HLtd -E ".git" -E "Library" . "$1" }
	fi

	source "$HOME/.fzf/shell/key-bindings.zsh"  # Source key bindings
fi


################################################################################
# Startup
################################################################################
# ------------------------------ Vimify Everything ----------------------------
set -o vi  # Vi mode
bindkey -M vicmd v edit-command-line  # Hitting `v` in escape mode opens command in Vim
export MANPAGER="vim -M +MANPAGER --not-a-term -"

# ----------------------------------- ZSH Rules -------------------------------
setopt inc_append_history_time  # Put time in history so we can see how long it takes things to run
source $HOME/.aliases
source $HOME/.functions

# ---------------------------- Launch tmux on start ---------------------------
if [[ `which tmux &>/dev/null && echo "$?"` -eq 0 ]]; then
	# First session named "main". Others named alt0, alt1, alt2, ...
	if [[ -z "$TMUX" ]]; then
		LASTSESSION=`tmux ls 2>/dev/null | awk -F":" '{print $1}' | sort | tail -2 | head -1`
		if [[ $LASTSESSION == main ]]; then
			tmux new -s alt0
		elif [[ $LASTSESSION == alt* ]]; then
			NUMBER=$(( 10#$(echo $LASTSESSION | grep -Eo '\d+') + 1 ))
			tmux new -s alt$NUMBER
		else
			tmux new -s main
		fi
	fi
fi
