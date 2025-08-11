document.addEventListener('DOMContentLoaded', function() {
    // Add a timestamp to show when the data was last updated
    const now = new Date();
    const footer = document.querySelector('footer');

    if (footer) {
        const lastUpdated = document.createElement('p');
        lastUpdated.textContent = `Last updated: ${now.toLocaleString()}`;
        lastUpdated.style.fontSize = '0.85rem';
        lastUpdated.style.marginTop = '0.5rem';
        footer.querySelector('.container').appendChild(lastUpdated);
    }

    // 保存原始仓库卡片，用于后续过滤操作
    window.originalRepoCards = Array.from(document.querySelectorAll('.repo-card')).map(card => card.cloneNode(true));
    console.log(`已保存 ${window.originalRepoCards.length} 个原始仓库卡片供过滤使用`);

    // 添加过滤功能 - 项目过滤
    const repoFilterBtns = document.querySelectorAll('#repositories .filter-btn');

    // 添加官方仓库列表 - 用于确保这些仓库在选择相应模型时总是显示
    const officialRepos = {
        'cursor': ['getcursor'],
        'deepseek': ['deepseek-ai'],
        'claude': ['anthropic'],
        'gemini': ['google-gemini'],
        'llama': ['meta-llama'],
        'qwen': ['QwenLM'],
        'hunyuan': ['HunyuanVideo', 'HunyuanDiT'],
    };

    // 用于存储从API获取的特定模型仓库
    let modelSpecificRepos = {};

    // 用于记录最后选择的主过滤器和模型过滤器
    let activeMainFilter = 'all';
    let activeModelFilter = '';

    // 处理下拉菜单中的模型过滤器点击
    const dropdownFilterBtns = document.querySelectorAll('#repositories .dropdown-content .filter-btn');
    dropdownFilterBtns.forEach(btn => {
        btn.addEventListener('click', function(e) {
            e.stopPropagation(); // 阻止事件冒泡，防止触发父元素的点击事件
            console.log(`点击了模型过滤器: ${this.textContent}`); // 调试信息

            // 移除所有主过滤器按钮的active类
            document.querySelectorAll('#repositories .section-actions > .filter-btn').forEach(b => {
                b.classList.remove('active');
            });

            // 获取过滤类型
            activeModelFilter = this.getAttribute('data-filter');
            console.log(`设置模型过滤器: ${activeModelFilter}`); // 调试信息

            activeMainFilter = ''; // 重置主过滤器，因为我们要专注于显示模型相关仓库

            // 更新下拉按钮文本显示当前选择的模型
            const dropdownBtn = document.querySelector('#repositories .dropbtn');
            if (dropdownBtn) {
                dropdownBtn.textContent = this.textContent + ' ▼';
                // 高亮显示下拉按钮
                dropdownBtn.classList.add('active');
            }

            // 从仓库中过滤特定模型的仓库
            fetchModelRepos(activeModelFilter);
        });
    });

    // 处理主过滤器点击
    const mainFilterBtns = document.querySelectorAll('#repositories .section-actions > .filter-btn');
    mainFilterBtns.forEach(btn => {
        btn.addEventListener('click', function() {
            console.log(`点击了主过滤器: ${this.textContent}`); // 调试信息

            // 移除所有主过滤器按钮的active类
            mainFilterBtns.forEach(b => b.classList.remove('active'));

            // 给当前按钮添加active类
            this.classList.add('active');

            // 获取过滤类型
            activeMainFilter = this.getAttribute('data-filter');
            console.log(`设置主过滤器: ${activeMainFilter}`); // 调试信息

            // 重置模型过滤器
            activeModelFilter = '';
            const dropdownBtn = document.querySelector('#repositories .dropbtn');
            if (dropdownBtn) {
                dropdownBtn.textContent = 'Models ▼';
                dropdownBtn.classList.remove('active');
            }

            // 清除特定模型的仓库缓存
            modelSpecificRepos = {};

            // 重置repo-grid，移除任何错误消息
            const repoGrid = document.querySelector('.repo-grid');
            if (repoGrid) {
                // 如果有"error"或"empty-state"类的元素，则清空列表
                if (repoGrid.querySelector('.error') || repoGrid.querySelector('.empty-state')) {
                    repoGrid.innerHTML = '';
                    // 重新显示所有卡片
                    document.querySelectorAll('.repo-card').forEach(card => {
                        card.style.display = 'block';
                    });
                }
            }

            // 应用过滤
            applyRepoFilters();
        });
    });

    // 从已有的仓库卡片中过滤特定模型的仓库
    function fetchModelRepos(modelName) {
        if (!modelName) {
            console.error('No model name provided to fetchModelRepos');
            return;
        }

        console.log(`开始查找 ${modelName} 相关仓库...`); // 调试信息

        // 显示加载状态
        const repoGrid = document.querySelector('.repo-grid');
        if (repoGrid) {
            repoGrid.innerHTML = '<div class="loading"><i class="fas fa-spinner fa-spin"></i><p>正在查找相关仓库...</p></div>';
        }

        try {
            // 使用保存的原始仓库卡片进行过滤，而不是当前页面上可能已经过滤过的卡片
            const allRepoCards = window.originalRepoCards || document.querySelectorAll('.repo-card');
            console.log(`总共找到 ${allRepoCards.length} 个仓库卡片`); // 调试信息

            if (allRepoCards.length === 0) {
                console.error('未找到任何仓库卡片，将刷新页面');
                // 如果没有找到仓库卡片，尝试刷新页面
                window.location.reload();
                return;
            }

            // 先清空当前网格，以便添加过滤后的结果
            repoGrid.innerHTML = '';

            // 记录匹配的仓库数量
            let visibleCount = 0;

            // 遍历所有卡片，查找匹配的
            allRepoCards.forEach(card => {
                // 提取仓库名称和描述
                const nameElem = card.querySelector('h3 a');
                const descElem = card.querySelector('.description');

                if (!nameElem) return; // 跳过没有名称的卡片

                const fullName = nameElem.textContent.trim();
                const name = fullName.toLowerCase();
                const description = descElem ? descElem.textContent.toLowerCase() : '';

                console.log(`检查仓库: ${fullName}`); // 调试信息

                // 判断是否匹配当前模型
                let isMatch = false;

                // 特殊情况：检查是否是官方仓库
                if (modelName.toLowerCase() === 'gemini' && name.includes('google-gemini')) {
                    isMatch = true;
                    console.log(`特殊匹配 Gemini: ${fullName}`);
                }
                else if (modelName.toLowerCase() === 'llama' && (name.includes('meta-llama') || name.includes('llama-cookbook'))) {
                    isMatch = true;
                    console.log(`特殊匹配 Llama: ${fullName}`);
                }
                else if (officialRepos[modelName]) {
                    // 检查是否匹配官方组织
                    const isOfficial = officialRepos[modelName].some(org =>
                        name.includes(org.toLowerCase())
                    );

                    if (isOfficial) {
                        isMatch = true;
                        console.log(`官方仓库匹配: ${fullName}`);
                    }
                }

                // 如果不是官方仓库，检查名称和描述是否包含模型名
                if (!isMatch) {
                    const modelNameLower = modelName.toLowerCase();

                    // 名称中包含模型名
                    if (name.includes(modelNameLower)) {
                        isMatch = true;
                        console.log(`名称匹配: ${fullName} 包含 ${modelName}`);
                    }
                    // 描述中包含模型名
                    else if (description && description.includes(modelNameLower)) {
                        isMatch = true;
                        console.log(`描述匹配: ${fullName} 描述中包含 ${modelName}`);
                    }
                }

                // 如果匹配，将卡片添加到结果中
                if (isMatch) {
                    visibleCount++;

                    // 创建卡片的深度克隆
                    const clonedCard = card.cloneNode(true);

                    // 如果是官方仓库且尚未添加官方标记，添加官方标记
                    if (officialRepos[modelName] && officialRepos[modelName].some(org => name.includes(org.toLowerCase()))) {
                        if (!clonedCard.querySelector('.official-badge')) {
                            const cardTitle = clonedCard.querySelector('h3');
                            if (cardTitle) {
                                const badge = document.createElement('span');
                                badge.className = 'official-badge';
                                badge.textContent = 'Official';
                                cardTitle.appendChild(badge);
                            }
                        }
                    }

                    // 添加到网格
                    repoGrid.appendChild(clonedCard);
                }
            });

            console.log(`找到 ${visibleCount} 个匹配 ${modelName} 的仓库`);

            // 更新标题以显示当前显示的是哪个模型的仓库
            const filterTitle = document.querySelector('.repo-filter-title');
            if (filterTitle) {
                filterTitle.textContent = `${modelName} Repositories (${visibleCount})`;
                filterTitle.style.display = 'block';
            }

            // 更新过滤结果计数
            updateFilterResultCount(visibleCount);

            // 如果没有找到匹配的仓库，显示空状态
            if (visibleCount === 0) {
                repoGrid.innerHTML = `<div class="empty-state">
                    <i class="fas fa-search"></i>
                    <p>没有找到与 ${modelName} 相关的仓库</p>
                </div>`;
            }

        } catch (error) {
            console.error('过滤仓库时发生错误:', error);

            if (repoGrid) {
                repoGrid.innerHTML = `<div class="error">
                    <i class="fas fa-exclamation-triangle"></i>
                    <p>过滤仓库时发生错误: ${error.message}</p>
                    <button id="retry-btn" class="retry-button">重试</button>
                </div>`;

                // 添加重试按钮逻辑
                const retryBtn = document.getElementById('retry-btn');
                if (retryBtn) {
                    retryBtn.addEventListener('click', function() {
                        fetchModelRepos(modelName);
                    });
                }
            }
        }
    }

    // 检查是否是官方仓库
    function checkIfOfficialRepo(name, description, modelName) {
        // 安全检查：确保所有参数都存在
        if (!officialRepos[modelName] || !modelName || !name) {
            return false;
        }

        // 转换为小写以进行不区分大小写的比较
        const nameLower = name.toLowerCase();
        const modelNameLower = modelName.toLowerCase();
        const descriptionLower = description ? description.toLowerCase() : '';

        // 获取当前模型的官方组织/用户名列表
        const officialOrgs = officialRepos[modelName];

        // 直接检查特殊情况
        if (modelNameLower === 'gemini' && nameLower.includes('google-gemini')) {
            console.log(`官方仓库匹配(特殊规则): ${name}`); // 调试日志
            return true;
        }

        // 1. 检查是否由官方组织发布
        const isFromOfficialOrg = officialOrgs.some(org => {
            const orgLower = org.toLowerCase();
            const matchResult =
                nameLower.startsWith(orgLower + '/') ||  // 匹配完整的组织名称前缀
                nameLower.includes('/' + orgLower) ||    // 匹配组织名称作为路径的一部分
                nameLower === orgLower;                  // 完全匹配组织名称

            if (matchResult) {
                console.log(`官方组织匹配: ${name} 包含 ${org}`); // 调试日志
            }

            return matchResult;
        });

        if (isFromOfficialOrg) {
            return true;
        }

        // 2. 检查仓库路径是否包含模型名称 (如 meta-llama/llama-cookbook)
        const repoContainsModel =
            nameLower.includes(`${modelNameLower}`) ||
            nameLower.includes(`${modelNameLower.replace('-', '')}`);

        // 3. 检查描述中是否明确提到这是官方仓库
        const isDescribedAsOfficial =
            descriptionLower.includes(`official ${modelNameLower}`) ||
            descriptionLower.includes(`${modelNameLower} official`) ||
            descriptionLower.includes(`official implementation`) ||
            descriptionLower.includes(`official repository`);

        const result = isFromOfficialOrg || (repoContainsModel && isDescribedAsOfficial);

        if (result) {
            console.log(`官方仓库匹配(规则2&3): ${name}`); // 调试日志
        }

        return result;
    }

    // 应用仓库过滤器
    function applyRepoFilters() {
        // 使用活动的主过滤器和模型过滤器
        const mainFilter = activeMainFilter; // 'all', 'trending', 或 'official'
        const modelFilter = activeModelFilter; // 可能为空或特定模型名称

        console.log(`应用过滤器: 主过滤器=${mainFilter}, 模型过滤器=${modelFilter}`); // 调试信息

        // 显示"正在过滤"提示
        const repoGrid = document.querySelector('.repo-grid');
        if (repoGrid) {
            repoGrid.innerHTML = '<div class="loading"><i class="fas fa-spinner fa-spin"></i><p>正在应用过滤器...</p></div>';
        }

        // 应用模型过滤器优先
        if (modelFilter) {
            console.log(`开始应用模型过滤器: ${modelFilter}`); // 调试信息
            // 如果有模型过滤器，优先应用它
            fetchModelRepos(modelFilter);
            return; // 模型过滤器处理了所有显示，所以我们在这里返回
        }

        console.log(`应用主过滤器: ${mainFilter}`); // 调试信息

        // 如果没有模型过滤器，处理主过滤器
        // 使用保存的原始仓库卡片数据
        const allRepoCards = window.originalRepoCards || document.querySelectorAll('.repo-card');
        let visibleCount = 0;

        // 清除以前的标题
        const filterTitle = document.querySelector('.repo-filter-title');
        if (filterTitle) {
            filterTitle.textContent = '';
            filterTitle.style.display = 'none';
        }

        // 清空网格准备重新填充
        if (repoGrid) {
            repoGrid.innerHTML = '';
        }

        // 应用主过滤器
        allRepoCards.forEach(card => {
            // 默认假设该卡片应该显示
            let shouldShow = true;

            // 根据主过滤器检查是否应该显示
            if (mainFilter !== 'all') {
                // 检查是否匹配主过滤器 (非'all'情况)
                const nameElem = card.querySelector('h3 a');
                const descElem = card.querySelector('.description');

                if (!nameElem) {
                    shouldShow = false;
                } else {
                    const name = nameElem.textContent.toLowerCase();
                    const desc = descElem ? descElem.textContent.toLowerCase() : '';

                    // 主过滤器检查
                    if (mainFilter === 'llm' && (name.includes('llm') || desc.includes('llm') ||
                                               name.includes('language model') || desc.includes('language model'))) {
                        shouldShow = true;
                    } else if (mainFilter === 'agent' && (name.includes('agent') || desc.includes('agent'))) {
                        shouldShow = true;
                    } else if (mainFilter === 'multimodal' && (name.includes('multimodal') || desc.includes('multimodal') ||
                                                            name.includes('multi-modal') || desc.includes('multi-modal'))) {
                        shouldShow = true;
                    } else if (mainFilter === 'diffusion' && (name.includes('diffusion') || desc.includes('diffusion'))) {
                        shouldShow = true;
                    } else {
                        shouldShow = false;
                    }
                }
            }

            // 将匹配的卡片添加到网格
            if (shouldShow) {
                visibleCount++;
                const clone = card.cloneNode(true);
                repoGrid.appendChild(clone);
            }
        });

        console.log(`主过滤器匹配结果: ${visibleCount} 个仓库`); // 调试信息

        // 更新过滤结果计数
        updateFilterResultCount(visibleCount);

        // 如果没有可见的卡片，显示空状态
        if (visibleCount === 0) {
            repoGrid.innerHTML = `<div class="empty-state">
                <i class="fas fa-search"></i>
                <p>没有找到匹配当前过滤器的仓库</p>
            </div>`;
        }
    }

    // 更新过滤结果计数
    function updateFilterResultCount(count) {
        const countElement = document.querySelector('.filter-result-count');
        if (countElement) {
            countElement.textContent = count === 1 ? '1 repository' : `${count} repositories`;
        }
    }

    // 添加论文过滤功能
    const paperFilterBtns = document.querySelectorAll('#papers .filter-btn');

    paperFilterBtns.forEach(btn => {
        btn.addEventListener('click', function() {
            // 移除所有按钮的active类
            paperFilterBtns.forEach(b => b.classList.remove('active'));
            // 给当前按钮添加active类
            this.classList.add('active');

            // 获取过滤类型
            const filter = this.getAttribute('data-filter');

            // 获取所有论文卡片
            const paperCards = document.querySelectorAll('.paper-card');

            // 过滤论文卡片
            paperCards.forEach(card => {
                if (filter === 'all') {
                    card.style.display = 'block';
                } else {
                    const title = card.querySelector('h3 a').textContent.toLowerCase();
                    const summary = card.querySelector('.summary')?.textContent.toLowerCase() || '';
                    const keywords = Array.from(card.querySelectorAll('.keyword'))
                        .map(k => k.textContent.toLowerCase());

                    if (filter === 'llm' &&
                        (title.includes('llm') ||
                         title.includes('language model') ||
                         summary.includes('language model') ||
                         keywords.some(k => k.includes('llm') || k.includes('language model')))) {
                        card.style.display = 'block';
                    } else if (filter === 'agent' &&
                        (title.includes('agent') ||
                         summary.includes('agent') ||
                         keywords.some(k => k.includes('agent')))) {
                        card.style.display = 'block';
                    } else if (filter === 'multimodal' &&
                        (title.includes('multimodal') ||
                         summary.includes('multimodal') ||
                         summary.includes('multi-modal') ||
                         keywords.some(k => k.includes('multimodal') || k.includes('multi-modal')))) {
                        card.style.display = 'block';
                    } else if (filter === 'diffusion' &&
                        (title.includes('diffusion') ||
                         summary.includes('diffusion') ||
                         keywords.some(k => k.includes('diffusion')))) {
                        card.style.display = 'block';
                    } else {
                        card.style.display = 'none';
                    }
                }
            });
        });
    });

    // Add click event to repository cards for better mobile experience
    const repoCards = document.querySelectorAll('.repo-card');
    repoCards.forEach(card => {
        card.addEventListener('click', function(e) {
            // Only trigger if the click wasn't on the link itself
            if (!e.target.closest('a')) {
                const link = card.querySelector('h3 a');
                if (link) {
                    window.open(link.href, '_blank');
                }
            }
        });
    });
});