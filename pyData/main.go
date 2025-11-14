package main

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const (
	DB_HOST     = "43.138.113.105"
	DB_PORT     = 13306
	DB_USER     = "root"
	DB_PASSWORD = "tKnrfzLkDe5nKbYP"
	DB_NAME     = "hub"
)

var (
	USER_COUNT          = 120000 // 用户数量
	ARTICLE_COUNT       = 24000  // 文章数量
	RESOURCE_COUNT      = 15000  // 资源数量
	COMMENT_COUNT       = 320000 // 评论数量
	CHAT_MESSAGE_COUNT  = 450000 // 聊天信息数量
	LIKE_COUNT          = 600000 // 点赞数量
	LOGIN_HISTORY_COUNT = 180000 // 登录历史数量
	STATISTICS_COUNT    = 3650   // 统计数据天数
)

var workerCount = determineWorkerCount()

// 添加一个全局的用户名映射来确保用户名唯一性
var usedUsernames = make(map[string]bool)
var usernameMutex sync.Mutex

type articleTopic struct {
	Title    string
	Summary  string
	Sections []string
	CTA      string
}

type resourceSeed struct {
	Title       string
	Description string
	CategoryID  int
	FileType    string
	Extension   string
}

var familyNames = []string{
	"赵", "钱", "孙", "李", "周", "吴", "郑", "王", "冯", "陈", "褚", "卫",
	"蒋", "沈", "韩", "杨", "朱", "秦", "尤", "许", "何", "吕", "施", "张",
	"孔", "曹", "严", "华", "金", "魏", "陶", "姜", "戚", "谢", "邹", "喻",
	"柏", "水", "窦", "章", "云", "苏", "潘", "葛", "奚", "范", "彭", "郎",
	"鲁", "韦", "昌", "马", "苗", "凤", "花", "方", "俞", "任", "袁", "柳",
	"鲍", "史", "唐", "费", "廉", "岑", "薛", "雷", "贺", "倪", "汤", "滕",
	"殷", "罗", "毕", "郝", "邬", "安", "常", "乐", "于", "时", "傅", "皮",
	"卞", "齐", "康", "伍", "余", "元", "卜", "顾", "孟", "平", "黄", "和",
	"穆", "萧", "尹", "姚", "邵", "湛", "汪", "祁", "毛", "禹", "狄", "米",
	"贝", "明", "臧",
	// 添加更多姓氏
	"唐宇", "上官", "欧阳", "夏侯", "诸葛", "闻人", "东方", "赫连", "皇甫", "尉迟", "公羊", "澹台", "公冶", "宗政", "濮阳", "淳于", "单于", "太叔", "申屠", "公孙", "仲孙", "轩辕", "令狐", "钟离", "宇文", "长孙", "慕容", "鲜于", "闾丘", "司徒", "司空", "亓官", "司寇", "仉", "督", "子车", "颛孙", "端木", "巫马", "公西", "漆雕", "乐正", "壤驷", "公良", "拓跋", "夹谷", "宰父", "谷梁", "楚晋", "阎法", "汝鄢", "涂钦", "段干", "百里", "东郭", "南门", "呼延", "归海", "羊舌", "微生", "岳帅", "缑亢", "况郈", "有琴", "梁丘", "左丘", "东门", "西门",
}

var givenNames = []string{
	"晨", "昊", "宇", "航", "泽", "琪", "瑞", "博", "潇", "言", "依", "涵",
	"若", "一", "宸", "岚", "婧", "珂", "婷", "萌", "颢", "芮", "歌", "屿",
	"朵", "杉", "璇", "可", "灵", "峻", "岳", "夕", "淼", "潼", "浩", "烁",
	"禹", "辰", "语", "霏", "钰", "洋", "初", "皓", "婉", "祺", "勋", "沐",
	"渝", "恬", "逸", "南", "锦", "渊", "烨", "铭", "星", "澜", "珞", "澈",
	"语桐", "诗涵", "若溪", "景曜", "惟一", "安歌", "知远", "清野", "序章", "青禾", "明川", "海岚",
	// 添加更多名字
	"紫萱", "雨嘉", "梦洁", "雅琳", "婧琪", "婉儿", "雅静", "梦舒", "睿婕", "婧雯", "婧涵", "婧萱",
	"雅馨", "雅彤", "雅涵", "雅琪", "雅琳", "雅琴", "雅芙", "雅菲", "雅茹", "雅婷", "雅芸", "雅菡",
	"雅彤", "雅昕", "雅懿", "雅咏", "雅媛", "雅悦", "雅韵", "雅芝", "雅菁", "雅葳", "雅滢", "雅伊",
	"思雨", "思佳", "思彤", "思涵", "思怡", "思妍", "思婕", "思慧", "思颖", "思瑶", "思莹", "思瑜",
	"欣怡", "欣妍", "欣悦", "欣颖", "欣茹", "欣妍", "欣怡", "欣悦", "欣颖", "欣茹", "欣妍", "欣怡",
	"俊杰", "俊驰", "俊熙", "俊茂", "俊逸", "俊朗", "俊才", "俊哲", "俊睿", "俊德", "俊彦", "俊明",
	"伟宸", "伟泽", "伟祺", "伟诚", "伟毅", "伟志", "伟瀚", "伟茂", "伟诚", "伟祺", "伟泽", "伟宸",
	"修杰", "修远", "修然", "修文", "修雅", "修齐", "修明", "修能", "修伟", "修洁", "修谨", "修谨",
	"博文", "博涛", "博超", "博涛", "博睿", "博涛", "博超", "博睿", "博涛", "博超", "博睿", "博涛",
}

var englishHandles = []string{
	"nebula", "atlas", "aurora", "quartz", "pixel", "echo", "lumen", "orbit", "zenith", "vertex",
	"cosmos", "nova", "lighthouse", "evergreen", "sunrise", "render", "delta", "aurum", "fusion", "lattice",
	"arctic", "volt", "shader", "fluent", "polaris", "zeno", "solstice", "ranger", "pioneer", "horizon",
	"crystal", "afterglow", "lyric", "binary", "syntax", "momentum", "uplink", "aperture", "magneto", "helios",
	// 添加更多英文用户名
	"matrix", "quantum", "phoenix", "cipher", "serenity", "infinity", "vortex", "neon", "stellar", "cipher",
	"mirage", "cipher", "nebula", "phantom", "onyx", "obsidian", "ember", "crimson", "azure", "verdant",
	"cobalt", "titanium", "mercury", "platinum", "diamond", "ruby", "sapphire", "emerald", "topaz", "opal",
	"amethyst", "garnet", "quartz", "jade", "pearl", "coral", "ivory", "ebony", "crystal", "marble",
	"granite", "basalt", "obsidian", "flint", "steel", "iron", "bronze", "silver", "gold", "platinum",
	"titanium", "carbon", "silicon", "neon", "argon", "krypton", "xenon", "radon", "helium", "hydrogen",
	"oxygen", "nitrogen", "fluorine", "chlorine", "sulfur", "phosphorus", "iodine", "mercury", "lead", "tin",
	"copper", "zinc", "nickel", "cobalt", "platinum", "palladium", "rhodium", "iridium", "osmium", "ruthenium",
	"technetium", "promethium", "europium", "gadolinium", "terbium", "dysprosium", "holmium", "erbium", "thulium", "ytterbium",
	"lutetium", "hafnium", "tantalum", "tungsten", "rhenium", "osmium", "iridium", "platinum", "gold", "mercury",
}

var emailDomains = []string{
	"hub.local", "mailhub.cn", "post.example", "labtech.dev", "stackgo.io", "flowbyte.com", "infra.run", "zerops.cn",
	// 添加更多邮箱域名
	"tech.dev", "cloud.net", "data.io", "ai.ml", "devops.org", "container.run", "k8s.io", "microsvc.app",
	"serverless.fun", "distributed.systems", "bigdata.analytic", "machine.learning", "deep.learning", "neural.net",
	"blockchain.fin", "crypto.wallet", "virtual.reality", "augmented.world", "iot.devices", "edge.compute",
	"quantum.bits", "nano.tech", "bio.informatics", "genomic.data", "proteomics.research", "drug.discovery",
	"climate.model", "weather.forecast", "satellite.imaging", "space.exploration", "rocket.science", "astronomy.observatory",
}

var passwordCandidates = []string{
	"Passw0rd!", "GoLang#2024", "HubTest@123", "P@ssword_1", "L0calTest!", "Sup3rSecure", "TestData#42", "GoLangR0cks", "Sh@dowMode!", "DataMock2025",
	// 添加更多密码候选
	"MyS3cur3P@ss", "T3st1ng!2024", "D3v3l0p3r#Key", "Syst3m@Admin", "Us3rAcc3ss$2024", "Cl0udS3rv!ce", "D@t@B@s3P@ss", "WebApp#Secure2024",
	"ApiT0k3n!2024", "S3ssionK3y#Val", "Encr7pt10nK3y", "HashV@lu3$Salt", "Rand0m!Ch@rs", "Sp3c!@lCh@rs", "Num83r$ymb0ls", "Mix3dC@s3P@ss",
	"C0mpl3x!P@ss", "Str0ng#K3yV@l", "S3cur3Us3r!2024", "Auth3nt!c@t3", "V3r!fyUs3r#24", "Acc3ssGr@nt3d", "P3rm!ss!0nSet", "R0l3B@s3d@ccess",
	"MultiF@ct0r!2024", "Bi0m3tr!cK3y", "F@ceR3c0gn!t10n", "Fing3rPr!ntSc@n", "R3t1n@Sc@nK3y", "Iris$can#Data", "DN@S3qu3nc3", "Gen0m1cD@t@K3y",
}

var websiteSuffix = []string{"dev", "studio", "team", "space", "codes", "cloud", "app", "work", "zone", "lab",
	// 添加更多网站后缀
	"io", "tech", "ai", "ml", "data", "cloud", "net", "org", "systems", "platform", "services", "solutions",
	"enterprise", "business", "company", "corporate", "group", "holdings", "ventures", "capital", "partners",
	"consulting", "agency", "digital", "online", "web", "site", "website", "host", "server", "compute", "network",
}

var bioSnippets = []string{
	"热衷云原生与容器调度，喜欢写自动化脚本", "关注可视化体验，常分享前端笔记",
	"正在把 AI 能力接入社区产品，想让技术贴近人", "常驻开源仓库 review PR，坚持快速学习",
	"迷上性能诊断与可观测性，希望把指标讲清楚", "研究数据治理和 A/B 实验，希望让决策透明",
	"偏爱命令行世界，也乐于分享工具链", "周末会组织读书会，探讨工程文化",
	"最近在编写 SRE 手册，沉迷 incident 复盘", "热爱城市骑行，记录不同社区的协作故事",
	// 添加更多个人简介片段
	"专注于微服务架构设计，擅长系统性能优化", "热衷于DevOps实践，推动CI/CD流程改进",
	"深度参与开源社区，贡献多个知名项目", "专注于大数据处理，擅长实时数据分析",
	"热衷技术分享，定期在技术大会上演讲", "专注于安全领域，擅长渗透测试和漏洞挖掘",
	"热爱编程教育，致力于降低编程学习门槛", "专注于移动开发，对用户体验有独特见解",
	"热衷于区块链技术，关注去中心化应用发展", "专注于人工智能，研究机器学习算法优化",
	"热爱产品设计，关注用户需求与体验优化", "专注于物联网开发，擅长边缘计算解决方案",
	"热衷于技术写作，博客累计阅读量超百万", "专注于测试自动化，提升软件质量保障效率",
	"热爱技术管理，有丰富的团队建设经验", "专注于云计算架构，帮助企业实现数字化转型",
}

var interestFocus = []string{
	"低代码编排", "微服务治理", "AI 绘画", "IoT 数据接入", "SRE 实践", "Rust 语言", "All in Go", "设计系统",
	"后量子密码", "流批一体", "边缘计算", "数据安全", "可观测性平台", "数据中台", "GPU 调度", "多云容灾",
	// 添加更多兴趣关注点
	"Serverless架构", "无服务器计算", "容器编排", "Kubernetes实践", "云原生安全", "微服务监控",
	"API网关设计", "服务网格", "混沌工程", "灰度发布", "蓝绿部署", "A/B测试框架", "数据湖构建",
	"数据仓库优化", "实时数据处理", "流式计算", "批处理优化", "分布式事务", "一致性协议",
	"区块链应用", "智能合约开发", "DeFi项目", "NFT平台", "元宇宙技术", "VR/AR应用",
	"机器学习平台", "深度学习框架", "自然语言处理", "计算机视觉", "语音识别", "推荐系统",
	"搜索引擎优化", "全文检索", "图数据库", "时序数据库", "NewSQL数据库", "分布式存储",
	"对象存储", "文件系统", "缓存策略", "CDN优化", "网络协议", "TCP/IP优化", "HTTP/3",
	"QUIC协议", "WebSocket", "消息队列", "异步处理", "事件驱动", "响应式编程", "函数式编程",
	"响应式系统", "弹性伸缩", "自动故障恢复", "自愈系统", "智能运维", "AIOps", "MLOps",
	"DevSecOps", "GitOps", "基础设施即代码", "配置管理", "密钥管理", "零信任安全", "身份认证",
	"访问控制", "权限管理", "审计日志", "合规性检查", "数据隐私保护", "GDPR合规", "数据脱敏",
	"加密传输", "端到端加密", "数字签名", "证书管理", "PKI体系", "区块链共识", "智能DNS",
	"负载均衡", "反向代理", "正向代理", "网关设计", "防火墙策略", "入侵检测", "恶意软件防护",
}

var articleTopics = []articleTopic{
	{
		Title:   "云原生观测从零到一",
		Summary: "从指标、日志、链路三个角度拆解观测系统的搭建细节，并分享踩坑经验。",
		Sections: []string{
			"快速梳理指标体系，选择 Prometheus 的原因以及常见聚合写法。",
			"链路追踪落地前后的指标对比，如何平衡吞吐和成本。",
			"告警调优 checklist，包含噪声过滤、收敛和升级策略。",
		},
		CTA: "提供自研 SLO 模板和 Grafana 仪表盘导入脚本。",
	},
	{
		Title:   "大模型推理服务压测记要",
		Summary: "记录在 GPU 集群里搭建推理服务、压测以及调优的全过程。",
		Sections: []string{
			"30 分钟内准备两套可回滚的推理镜像及权重。",
			"针对 QPS 抖动的排查顺序：先 driver 再 Node，再排网络。",
			"压测监控项优先级，GPU 使用率其实排不到前三。",
		},
		CTA: "附上压测脚本与节点火焰图示例。",
	},
	{
		Title:   "Go 微服务灰度发布清单",
		Summary: "整理了一套在 Kubernetes 中发布 Go 微服务的灰度清单，可直接复用。",
		Sections: []string{
			"多版本探针与 readiness gate 的配置示例。",
			"二级灰度指标：异常率、P99、TPS 回放阈值。",
			"常见回滚路径与镜像、配置联动。",
		},
		CTA: "附带 release 管理器的开源地址。",
	},
	{
		Title:   "数据中台权限系统拆解",
		Summary: "从角色、项目、资源三层模型拆解复杂权限体系。",
		Sections: []string{
			"三层模型如何解耦与扩展。",
			"DSQL 权限校验链路，如何在 3ms 内完成跨库授权。",
			"审计日志写入策略以及避免热点的方法。",
		},
		CTA: "附带 ER 图与初始化脚本。",
	},
	{
		Title:   "协同代码编辑体验设计",
		Summary: "记录团队如何实现低延迟的协同代码编辑器，兼顾权限与审计。",
		Sections: []string{
			"OT 与 CRDT 的技术选型及实现差异。",
			"语法高亮与 LSP 插件的通信协议。",
			"冲突合并策略以及弱网降级方案。",
		},
		CTA: "开放 WebSocket 层压测脚本便于复现。",
	},
	{
		Title:   "API Gateway 性能诊断手记",
		Summary: "分享一次在高峰期抓包、定位、治理网关性能的经历。",
		Sections: []string{
			"如何快速还原慢请求的真实路径。",
			"多租户限流策略的动态调节。",
			"使用 eBPF 采样定位 CPU 异常。",
		},
		CTA: "包含 flamegraph 以及告警规则模板。",
	},
	{
		Title:   "A/B 实验平台落地实践",
		Summary: "从技术与流程两方面讲清楚上线一套实验平台的要点。",
		Sections: []string{
			"指标体系如何抽象才能兼容业务方差。",
			"样本数、实验权重、回滚策略的协同。",
			"对接 BI 与风控系统的注意事项。",
		},
		CTA: "附录提供埋点 SDK 与实验评估模板。",
	},
	// 添加更多文章主题
	{
		Title:   "Serverless 架构在企业级应用中的实践",
		Summary: "深入探讨 Serverless 架构在企业级应用中的实际应用和挑战。",
		Sections: []string{
			"Serverless 的成本模型分析与优化策略。",
			"冷启动问题的解决方案与性能调优。",
			"如何设计事件驱动的微服务架构。",
		},
		CTA: "提供完整的架构图和部署模板。",
	},
	{
		Title:   "基于 Kubernetes 的多云部署策略",
		Summary: "分享在多个云平台上部署 Kubernetes 集群的经验和最佳实践。",
		Sections: []string{
			"跨云平台的集群管理工具选型。",
			"多云环境下的网络策略和安全配置。",
			"灾难恢复和数据同步方案。",
		},
		CTA: "附带 Terraform 脚本和 Helm Charts。",
	},
	{
		Title:   "微服务架构下的数据一致性保障",
		Summary: "探讨在分布式微服务架构中如何保障数据一致性。",
		Sections: []string{
			"分布式事务的实现方案对比。",
			"最终一致性模型的设计与实现。",
			"数据冲突检测和解决机制。",
		},
		CTA: "提供示例代码和测试用例。",
	},
	{
		Title:   "DevSecOps 在企业中的落地实践",
		Summary: "介绍如何将安全集成到 DevOps 流程中，实现 DevSecOps。",
		Sections: []string{
			"安全左移的实施策略和工具链。",
			"自动化安全扫描和漏洞管理。",
			"安全合规性检查和报告生成。",
		},
		CTA: "提供安全检查清单和集成脚本。",
	},
	{
		Title:   "大规模容器集群的资源调度优化",
		Summary: "分享在大规模 Kubernetes 集群中优化资源调度的经验。",
		Sections: []string{
			"资源配额和限制的合理配置。",
			"调度器自定义和优化策略。",
			"集群资源利用率监控和分析。",
		},
		CTA: "附带调度器配置模板和监控仪表盘。",
	},
}

var resourceSeeds = []resourceSeed{
	{
		Title:       "Kubernetes 落地作战手册",
		Description: "涵盖集群规划、网络选型、运维巡检等 50+ 个实践，适合 Ops 团队快速搭建体系。",
		CategoryID:  1,
		FileType:    "application/pdf",
		Extension:   ".pdf",
	},
	{
		Title:       "Go 性能剖析案例库",
		Description: "收录多个真实项目的性能瓶颈案例，涵盖 CPU、内存、锁竞争。",
		CategoryID:  2,
		FileType:    "application/zip",
		Extension:   ".zip",
	},
	{
		Title:       "数据可视化组件库",
		Description: "包含丰富的 ECharts 与 AntV 组合示例，并配 Storybook 文档。",
		CategoryID:  3,
		FileType:    "application/vnd.ms-powerpoint",
		Extension:   ".pptx",
	},
	{
		Title:       "边缘计算网关固件",
		Description: "适配主流 ARM 设备的网关固件，集成 OTA、日志聚合与诊断工具。",
		CategoryID:  4,
		FileType:    "application/octet-stream",
		Extension:   ".img",
	},
	{
		Title:       "AIGC 创作者工作流模板",
		Description: "从提示词管理、素材产出到审核排期的一整套流程。",
		CategoryID:  5,
		FileType:    "application/vnd.ms-excel",
		Extension:   ".xlsx",
	},
	{
		Title:       "SaaS 安全评估清单",
		Description: "列出渗透、加密、合规、备份等关键项，帮助团队自查安全。",
		CategoryID:  2,
		FileType:    "text/plain",
		Extension:   ".md",
	},
	// 添加更多资源种子
	{
		Title:       "微服务架构设计模式",
		Description: "详细介绍了微服务架构中的各种设计模式及其应用场景。",
		CategoryID:  1,
		FileType:    "application/pdf",
		Extension:   ".pdf",
	},
	{
		Title:       "Docker 容器最佳实践",
		Description: "Docker 容器化应用的完整指南，包括镜像优化、网络安全等。",
		CategoryID:  1,
		FileType:    "application/pdf",
		Extension:   ".pdf",
	},
	{
		Title:       "Kubernetes Operator 开发框架",
		Description: "用于开发 Kubernetes Operator 的完整框架和示例代码。",
		CategoryID:  2,
		FileType:    "application/zip",
		Extension:   ".zip",
	},
	{
		Title:       "云原生安全防护手册",
		Description: "涵盖云原生环境下各种安全威胁的防护策略和工具。",
		CategoryID:  4,
		FileType:    "application/pdf",
		Extension:   ".pdf",
	},
	{
		Title:       "DevOps 流水线模板集合",
		Description: "包含多种编程语言和框架的 CI/CD 流水线模板。",
		CategoryID:  3,
		FileType:    "application/zip",
		Extension:   ".zip",
	},
	{
		Title:       "大数据处理工具箱",
		Description: "Apache Spark、Flink 等大数据处理框架的实用工具和脚本。",
		CategoryID:  5,
		FileType:    "application/zip",
		Extension:   ".zip",
	},
}

var commentTemplates = []string{
	"这一段写得太细了，收藏慢慢啃。",
	"试了下文中的脚本，确实能把耗时压到个位数。",
	"之前踩过类似的坑，现在终于知道正确姿势了。",
	"期待补充更多真实指标截图，方便对照。",
	"已经推荐给团队同事，一起学习。",
	"想听听你们在生产环境的阈值如何设定？",
	"作者真的很懂业务痛点，案例接地气。",
	"如果能附上 demo 仓库就更好了。",
	"写得这么细，可以直接当上线 checklist。",
	"希望后续能聊聊自动化回滚的细节。",
	// 添加更多评论模板
	"非常实用的技术分享，学到了很多。",
	"这个方案在我们项目中也适用，感谢分享。",
	"有没有考虑过在其他场景下的应用？",
	"文章结构清晰，内容详实，点赞！",
	"希望能看到更多类似的技术深度文章。",
	"有些地方还不太明白，能否详细解释一下？",
	"这个优化效果真的很明显，值得尝试。",
	"作者的实践经验很丰富，受益匪浅。",
	"在实际部署中遇到了一些问题，希望交流。",
	"这种解决方案比我们之前的方案优雅多了。",
	"感谢提供这么详细的步骤和代码示例。",
	"在实施过程中发现了一些需要注意的地方。",
	"这个思路很新颖，开拓了视野。",
	"对于初学者来说非常友好，容易理解。",
	"期待作者的下一篇文章，会持续关注。",
	"能否提供一下相关工具的下载链接？",
	"这个方法在我们的环境中也有效，谢谢！",
	"文章中的图表很清晰，有助于理解。",
	"有一些细节可能需要根据实际情况调整。",
	"这种架构设计思想很值得借鉴和学习。",
}

var messageSnippets = []string{
	"我们在 staging 环境复现出来了，正在排查配置。",
	"下周社群活动的嘉宾已确认，稍后发预告。",
	"谁有空帮忙看下这个 SQL，explain 有点奇怪。",
	"帮你同步到产研群里，等反馈再更新。",
	"记得提 Merge Request，review 排期已经塞满。",
	"最新的视觉稿在 Figma，请大家评审。",
	"CI 跑红了，可能是上游依赖升级，晚上一起看。",
	"大屏 metrics 又炸了，先把告警 muted 一下。",
	"客户问的 license 问题我先兜着，你专心查 bug。",
	"今天 18:30 做一次小范围上线，注意 watch 日志。",
	// 添加更多消息片段
	"新版本的部署脚本已经测试通过，可以合入主干。",
	"线上监控发现异常流量，正在分析原因。",
	"数据库连接池出现瓶颈，需要调整配置参数。",
	"API 响应时间增长了 30%，正在定位性能瓶颈。",
	"服务器磁盘空间不足，需要清理日志文件。",
	"新功能上线后用户反馈良好，数据表现不错。",
	"安全扫描发现潜在漏洞，需要紧急修复。",
	"第三方服务出现不稳定，正在寻找替代方案。",
	"用户权限系统有 Bug，紧急修复版本已发布。",
	"系统升级完成后性能提升明显，符合预期。",
	"新加入的同事需要培训，谁有时间帮忙带一下？",
	"本周的代码评审安排在周五下午，请提前准备。",
	"线上环境配置有变更，请注意同步到测试环境。",
	"客户提出了新需求，下周会议讨论实现方案。",
	"团队建设活动计划在下个月初，请大家投票选择。",
	"新的开发工具链已经配置好，大家可以开始使用。",
	"项目里程碑达成，感谢大家的努力和付出。",
	"线上故障复盘会议安排在明天上午，请准时参加。",
	"技术分享会本周五举行，主题是微服务架构演进。",
	"新招聘的岗位需求已发布，欢迎推荐合适人选。",
}

var loginUserAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 13_6) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0 Safari/537.36",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 16_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148",
	"Mozilla/5.0 (Linux; Android 14; Pixel 8 Pro) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0 Mobile Safari/537.36",
	// 添加更多User Agents
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:124.0) Gecko/20100101 Firefox/124.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:124.0) Gecko/20100101 Firefox/124.0",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 17_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (Linux; Android 13; SM-G981B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Mobile Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Edge/124.0.2478.80",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 13_6) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Safari/605.1.15",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36",
	"Mozilla/5.0 (iPad; CPU OS 16_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.4 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (Linux; Android 12; Pixel 6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Mobile Safari/537.36",
}

var provinceCities = map[string][]string{
	"上海市":      {"上海市"},
	"云南省":      {"临沧市", "丽江市", "保山市", "大理白族自治州", "德宏傣族景颇族自治州", "怒江傈僳族自治州", "文山壮族苗族自治州", "昆明市", "昭通市", "普洱市", "曲靖市", "楚雄彝族自治州", "玉溪市", "红河哈尼族彝族自治州", "西双版纳傣族自治州", "迪庆藏族自治州"},
	"内蒙古自治区":   {"乌兰察布市", "乌海市", "兴安盟", "包头市", "呼伦贝尔市", "呼和浩特市", "巴彦淖尔市", "赤峰市", "通辽市", "鄂尔多斯市", "锡林郭勒盟", "阿拉善盟"},
	"北京市":      {"北京市"},
	"台湾省":      {"云林县", "南投县", "台东县", "台中市", "台北市", "台南市", "嘉义县", "嘉义市", "基隆市", "宜兰县", "屏东县", "彰化县", "新北市", "新竹县", "新竹市", "桃园市", "澎湖县", "花莲县", "苗栗县", "连江县", "金门县", "高雄市"},
	"吉林省":      {"吉林市", "四平市", "延边朝鲜族自治州", "松原市", "白城市", "白山市", "辽源市", "通化市", "长春市"},
	"四川省":      {"乐山市", "内江市", "凉山彝族自治州", "南充市", "宜宾市", "巴中市", "广元市", "广安市", "德阳市", "成都市", "攀枝花市", "泸州市", "甘孜藏族自治州", "眉山市", "绵阳市", "自贡市", "资阳市", "达州市", "遂宁市", "阿坝藏族羌族自治州", "雅安市"},
	"天津市":      {"天津市"},
	"宁夏回族自治区":  {"中卫市", "吴忠市", "固原市", "石嘴山市", "银川市"},
	"安徽省":      {"亳州市", "六安市", "合肥市", "安庆市", "宣城市", "宿州市", "池州市", "淮北市", "淮南市", "滁州市", "芜湖市", "蚌埠市", "铜陵市", "阜阳市", "马鞍山市", "黄山市"},
	"山东省":      {"东营市", "临沂市", "威海市", "德州市", "日照市", "枣庄市", "泰安市", "济南市", "济宁市", "淄博市", "滨州市", "潍坊市", "烟台市", "聊城市", "菏泽市", "青岛市"},
	"山西省":      {"临汾市", "吕梁市", "大同市", "太原市", "忻州市", "晋中市", "晋城市", "朔州市", "运城市", "长治市", "阳泉市"},
	"广东省":      {"东莞市", "中山市", "云浮市", "佛山市", "广州市", "惠州市", "揭阳市", "梅州市", "汕头市", "汕尾市", "江门市", "河源市", "深圳市", "清远市", "湛江市", "潮州市", "珠海市", "肇庆市", "茂名市", "阳江市", "韶关市"},
	"广西壮族自治区":  {"北海市", "南宁市", "崇左市", "来宾市", "柳州市", "桂林市", "梧州市", "河池市", "玉林市", "百色市", "贵港市", "贺州市", "钦州市", "防城港市"},
	"新疆维吾尔自治区": {"乌鲁木齐市", "伊犁哈萨克自治州", "克孜勒苏柯尔克孜自治州", "克拉玛依市", "博尔塔拉蒙古自治州", "吐鲁番市", "和田地区", "哈密市", "喀什地区", "塔城地区", "巴音郭楞蒙古自治州", "昌吉回族自治州", "自治区直辖县级行政区划", "阿克苏地区", "阿勒泰地区"},
	"江苏省":      {"南京市", "南通市", "宿迁市", "常州市", "徐州市", "扬州市", "无锡市", "泰州市", "淮安市", "盐城市", "苏州市", "连云港市", "镇江市"},
	"江西省":      {"上饶市", "九江市", "南昌市", "吉安市", "宜春市", "抚州市", "新余市", "景德镇市", "萍乡市", "赣州市", "鹰潭市"},
	"河北省":      {"保定市", "唐山市", "廊坊市", "张家口市", "承德市", "沧州市", "石家庄市", "秦皇岛市", "衡水市", "邢台市", "邯郸市"},
	"河南省":      {"三门峡市", "信阳市", "南阳市", "周口市", "商丘市", "安阳市", "平顶山市", "开封市", "新乡市", "洛阳市", "漯河市", "濮阳市", "焦作市", "省直辖县级行政区划", "许昌市", "郑州市", "驻马店市", "鹤壁市"},
	"浙江省":      {"丽水市", "台州市", "嘉兴市", "宁波市", "杭州市", "温州市", "湖州市", "绍兴市", "舟山市", "衢州市", "金华市"},
	"海南省":      {"三亚市", "三沙市", "儋州市", "海口市", "省直辖县级行政区划"},
	"湖北省":      {"十堰市", "咸宁市", "孝感市", "宜昌市", "恩施土家族苗族自治州", "武汉市", "省直辖县级行政区划", "荆州市", "荆门市", "襄阳市", "鄂州市", "随州市", "黄冈市", "黄石市"},
	"湖南省":      {"娄底市", "岳阳市", "常德市", "张家界市", "怀化市", "株洲市", "永州市", "湘潭市", "湘西土家族苗族自治州", "益阳市", "衡阳市", "邵阳市", "郴州市", "长沙市"},
	"澳门特别行政区":  {"圣安多尼堂区", "大堂区", "望德堂区", "氹仔", "花地玛堂区", "路环", "风顺堂区"},
	"甘肃省":      {"临夏回族自治州", "兰州市", "嘉峪关市", "天水市", "定西市", "平凉市", "庆阳市", "张掖市", "武威市", "甘南藏族自治州", "白银市", "酒泉市", "金昌市", "陇南市"},
	"福建省":      {"三明市", "南平市", "厦门市", "宁德市", "泉州市", "漳州市", "福州市", "莆田市", "龙岩市"},
	"西藏自治区":    {"山南市", "拉萨市", "日喀则市", "昌都市", "林芝市", "那曲市", "阿里地区"},
	"贵州省":      {"六盘水市", "安顺市", "毕节市", "贵阳市", "遵义市", "铜仁市", "黔东南苗族侗族自治州", "黔南布依族苗族自治州", "黔西南布依族苗族自治州"},
	"辽宁省":      {"丹东市", "大连市", "抚顺市", "朝阳市", "本溪市", "沈阳市", "盘锦市", "营口市", "葫芦岛市", "辽阳市", "铁岭市", "锦州市", "阜新市", "鞍山市"},
	"重庆市":      {"重庆市"},
	"陕西省":      {"咸阳市", "商洛市", "安康市", "宝鸡市", "延安市", "榆林市", "汉中市", "渭南市", "西安市", "铜川市"},
	"青海省":      {"果洛藏族自治州", "海东市", "海北藏族自治州", "海南藏族自治州", "海西蒙古族藏族自治州", "玉树藏族自治州", "西宁市", "黄南藏族自治州"},
	"香港特别行政区":  {"九龙", "新界", "香港岛"},
	"黑龙江省":     {"七台河市", "伊春市", "佳木斯市", "双鸭山市", "哈尔滨市", "大兴安岭地区", "大庆市", "牡丹江市", "绥化市", "鸡西市", "鹤岗市", "黑河市", "齐齐哈尔市"},
}

// 定义省份权重，使分布更接近真实情况
var provinceWeights = map[string]int{
	"广东省":      8,
	"山东省":      7,
	"河南省":      7,
	"四川省":      6,
	"江苏省":      6,
	"河北省":      5,
	"湖南省":      5,
	"安徽省":      5,
	"湖北省":      5,
	"浙江省":      5,
	"广西壮族自治区":  4,
	"云南省":      4,
	"江西省":      4,
	"辽宁省":      4,
	"黑龙江省":     4,
	"陕西省":      4,
	"山西省":      4,
	"福建省":      4,
	"贵州省":      3,
	"重庆市":      3,
	"吉林省":      3,
	"甘肃省":      3,
	"内蒙古自治区":   3,
	"新疆维吾尔自治区": 3,
	"上海市":      3,
	"北京市":      3,
	"天津市":      2,
	"海南省":      2,
	"宁夏回族自治区":  2,
	"青海省":      1,
	"西藏自治区":    1,
	"香港特别行政区":  1,
	"澳门特别行政区":  1,
	"台湾省":      1,
}

// weightedRandomProvince 根据权重随机选择省份
func weightedRandomProvince(rnd *rand.Rand) string {
	// 计算总权重
	totalWeight := 0
	for _, weight := range provinceWeights {
		totalWeight += weight
	}

	// 生成随机数
	randomValue := rnd.Intn(totalWeight)

	// 根据权重选择省份
	currentWeight := 0
	for province, weight := range provinceWeights {
		currentWeight += weight
		if randomValue < currentWeight {
			return province
		}
	}

	// 如果未找到（理论上不应该发生），返回随机省份
	return randomChoice(rnd, provinceList)
}

var provinceList = []string{
	"上海市", "云南省", "内蒙古自治区", "北京市", "台湾省", "吉林省", "四川省", "天津市", "宁夏回族自治区", "安徽省", "山东省", "山西省", "广东省", "广西壮族自治区", "新疆维吾尔自治区", "江苏省", "江西省", "河北省", "河南省", "浙江省", "海南省", "湖北省", "湖南省", "澳门特别行政区", "甘肃省", "福建省", "西藏自治区", "贵州省", "辽宁省", "重庆市", "陕西省", "青海省", "香港特别行政区", "黑龙江省",
	// 添加更多省份
	"河北省2", "山西省2", "辽宁省2", "吉林省2", "黑龙江省2", "江苏省2", "浙江省2", "安徽省2", "福建省2", "江西省2", "山东省2", "河南省2", "湖北省2", "湖南省2", "广东省2", "海南省2", "四川省2", "贵州省2", "云南省2", "陕西省2", "甘肃省2", "青海省2", "台湾省2",
}

func determineWorkerCount() int {
	base := runtime.NumCPU()
	if base < 1 {
		base = 1
	}
	runtime.GOMAXPROCS(base)

	if override := strings.TrimSpace(os.Getenv("WORKER_COUNT")); override != "" {
		if parsed, err := strconv.Atoi(override); err == nil && parsed > 0 {
			if parsed > base {
				runtime.GOMAXPROCS(parsed)
			}
			return parsed
		}
	}

	workers := base * 64
	if workers < 8 {
		workers = 8
	}
	return workers
}

func runWorkers(total int, workerCount int, fn func(idx int, rnd *rand.Rand)) {
	if total <= 0 {
		return
	}
	if workerCount <= 0 {
		workerCount = 1
	}

	jobs := make(chan int, workerCount*4)
	var wg sync.WaitGroup

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			rnd := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID*9973)))
			for idx := range jobs {
				fn(idx, rnd)
			}
		}(i)
	}

	for i := 0; i < total; i++ {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
}

func randomChoice[T any](rnd *rand.Rand, arr []T) T {
	if len(arr) == 0 {
		var zero T
		return zero
	}
	return arr[rnd.Intn(len(arr))]
}

func randomFullName(rnd *rand.Rand) string {
	return randomChoice(rnd, familyNames) + randomChoice(rnd, givenNames)
}

func randomEmail(username string, rnd *rand.Rand) string {
	return fmt.Sprintf("%s@%s", username, randomChoice(rnd, emailDomains))
}

func randomPhone(rnd *rand.Rand) string {
	prefixes := []string{"130", "131", "132", "133", "134", "135", "136", "137", "138", "139", "186", "187"}
	prefix := randomChoice(rnd, prefixes)
	return fmt.Sprintf("%s%08d", prefix, rnd.Intn(100000000))
}

func randomIP(rnd *rand.Rand) string {
	return fmt.Sprintf("%d.%d.%d.%d", 10+rnd.Intn(200), rnd.Intn(255), rnd.Intn(255), 1+rnd.Intn(254))
}

func randomPastTime(rnd *rand.Rand, maxDays int) time.Time {
	if maxDays <= 0 {
		return time.Now()
	}
	hours := rnd.Intn(maxDays*24 + 24)
	return time.Now().Add(-time.Duration(hours) * time.Hour)
}

func randomBio(rnd *rand.Rand) string {
	first := randomChoice(rnd, bioSnippets)
	second := randomChoice(rnd, interestFocus)
	return fmt.Sprintf("%s，近期关注 %s。", first, second)
}

func randomWebsite(username string, rnd *rand.Rand) string {
	return fmt.Sprintf("https://%s.%s", strings.ReplaceAll(username, "_", "-"), randomChoice(rnd, websiteSuffix))
}

func randomGithub(username string) string {
	handle := strings.ReplaceAll(username, ".", "")
	handle = strings.ReplaceAll(handle, "_", "-")
	return fmt.Sprintf("https://github.com/%s", handle)
}

func randomHex(rnd *rand.Rand, length int) string {
	const alphabet = "0123456789abcdef"
	builder := strings.Builder{}
	for i := 0; i < length; i++ {
		builder.WriteByte(alphabet[rnd.Intn(len(alphabet))])
	}
	return builder.String()
}

func randomArticleContent(topic articleTopic, rnd *rand.Rand) string {
	builder := strings.Builder{}
	builder.WriteString("# ")
	builder.WriteString(topic.Title)
	builder.WriteString("\n\n")
	builder.WriteString(topic.Summary)
	builder.WriteString("\n\n---\n")
	for idx, section := range topic.Sections {
		builder.WriteString(fmt.Sprintf("## 部分 %d\n%s\n\n", idx+1, section))
	}
	builder.WriteString("### 小结\n")
	builder.WriteString(topic.CTA)
	builder.WriteString("\n\n> 来自社区真实案例，欢迎留言讨论。\n")
	if rnd.Float64() < 0.4 {
		builder.WriteString("\n``bash\n")
		builder.WriteString("kubectl get pods --all-namespaces\n")
		builder.WriteString("```\n")
	}
	return builder.String()
}

func randomPasswordHash(rnd *rand.Rand) string {
	return hashPassword(randomChoice(rnd, passwordCandidates))
}

func maybeString(rnd *rand.Rand, probability float64, value string) interface{} {
	if rnd.Float64() < probability {
		return value
	}
	return nil
}

func maybeTime(rnd *rand.Rand, probability float64, value time.Time) interface{} {
	if rnd.Float64() < probability {
		return value
	}
	return nil
}

func fetchIDs(db *sql.DB, table string) []int64 {
	rows, err := db.Query(fmt.Sprintf("SELECT id FROM %s", table))
	if err != nil {
		log.Fatalf("查询 %s 失败: %v", table, err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			log.Fatalf("扫描 %s id 失败: %v", table, err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		log.Fatalf("遍历 %s 结果失败: %v", table, err)
	}
	return ids
}

func main() {
	fmt.Println("开始生成测试数据...")
	startTime := time.Now()

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=Local",
		DB_USER, DB_PASSWORD, DB_HOST, DB_PORT, DB_NAME)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(workerCount * 4)
	db.SetMaxIdleConns(workerCount * 2)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		log.Fatal("数据库连接测试失败:", err)
	}
	fmt.Printf("✓ 数据库连接成功，使用 %d 个并发 worker 写入\n", workerCount)

	generateUsers(db)
	generateArticles(db)
	generateResources(db)
	generateComments(db)
	generateChatMessages(db)
	generateLikes(db)
	generateLoginHistory(db)
	generateStatistics(db)

	fmt.Printf("\n=== 数据生成完成 ===\n")
	fmt.Printf("总耗时: %v\n", time.Since(startTime))
	fmt.Println("🎉 所有数据生成完成！")
}

// hashPassword 生成密码哈希
func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return fmt.Sprintf("%x", hash)
}

// generateUsers 生成用户数据
func generateUsers(db *sql.DB) {
	fmt.Println("\n开始生成用户数据...")
	startTime := time.Now()

	authStmt, err := db.Prepare(`INSERT INTO user_auth (username, password_hash, email, role, auth_status, account_status, 
                                      last_login_time, last_login_ip, failed_login_count, created_at, updated_at)
                VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		log.Fatalf("准备用户认证插入语句失败: %v", err)
	}
	defer authStmt.Close()

	profileStmt, err := db.Prepare(`INSERT INTO user_profile (user_id, nickname, bio, avatar_url, phone, gender, birthday, 
                                         province, city, website, github, created_at, updated_at)
                VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		log.Fatalf("准备用户资料插入语句失败: %v", err)
	}
	defer profileStmt.Close()

	runWorkers(USER_COUNT, workerCount, func(i int, rnd *rand.Rand) {
		// 生成唯一用户名
		var username string
		var result sql.Result
		var err error
		var userID int64

		createdAt := randomPastTime(rnd, 420)
		updatedAt := createdAt.Add(time.Duration(rnd.Intn(240)) * time.Hour)

		for {
			for {
				handle := fmt.Sprintf("%s_%04d", randomChoice(rnd, englishHandles), 1000+rnd.Intn(9000))
				username = strings.ToLower(handle)

				// 检查用户名是否已使用
				usernameMutex.Lock()
				if !usedUsernames[username] {
					usedUsernames[username] = true
					usernameMutex.Unlock()
					break
				}
				usernameMutex.Unlock()
			}

			passwordHash := randomPasswordHash(rnd)
			email := randomEmail(username, rnd)

			role := "user"
			if rnd.Float64() < 0.07 {
				role = "admin"
			}

			authStatus := 1
			if rnd.Float64() < 0.05 {
				authStatus = 0
			}

			accountStatus := 1
			statusRoll := rnd.Float64()
			if statusRoll < 0.05 {
				accountStatus = 0
			} else if statusRoll < 0.12 {
				accountStatus = 2
			}

			lastLogin := randomPastTime(rnd, 45)
			lastLoginIP := randomIP(rnd)
			failedLoginCount := rnd.Intn(7)

			result, err = authStmt.Exec(username, passwordHash, email, role, authStatus, accountStatus,
				maybeTime(rnd, 0.82, lastLogin), maybeString(rnd, 0.82, lastLoginIP), failedLoginCount, createdAt, updatedAt)
			if err != nil {
				// 检查是否是重复键错误
				if strings.Contains(err.Error(), "Duplicate entry") {
					// 如果是重复键错误，标记该用户名为未使用并重新生成
					usernameMutex.Lock()
					delete(usedUsernames, username)
					usernameMutex.Unlock()
					continue
				} else {
					log.Fatalf("插入用户认证信息失败: %v", err)
				}
			}
			break
		}

		userID, err = result.LastInsertId()
		if err != nil {
			log.Fatalf("获取用户ID失败: %v", err)
		}

		nickname := randomFullName(rnd)
		bioValue := maybeString(rnd, 0.75, randomBio(rnd))
		avatarValue := maybeString(rnd, 0.7, fmt.Sprintf("https://cdn.hub.local/avatar/%s.png", username))
		phoneValue := maybeString(rnd, 0.55, randomPhone(rnd))
		gender := rnd.Intn(3)
		birthday := time.Now().AddDate(-18-rnd.Intn(15), -rnd.Intn(12), -rnd.Intn(28))

		// 完全随机化所在地
		province := randomChoice(rnd, provinceList)
		city := randomChoice(rnd, provinceCities[province])
		website := maybeString(rnd, 0.45, randomWebsite(username, rnd))
		github := maybeString(rnd, 0.35, randomGithub(username))

		_, err = profileStmt.Exec(userID, nickname, bioValue, avatarValue, phoneValue, gender,
			maybeTime(rnd, 0.9, birthday), province, city, website, github, createdAt, updatedAt)
		if err != nil {
			log.Fatalf("插入用户资料失败: %v", err)
		}
	})

	fmt.Printf("✓ 用户数据生成完成，共 %d 条记录，耗时: %v\n", USER_COUNT, time.Since(startTime))
}

func generateArticles(db *sql.DB) {
	fmt.Println("\n开始生成文章数据...")
	startTime := time.Now()

	articleStmt, err := db.Prepare(`INSERT INTO articles (user_id, title, description, content, status, view_count, like_count, comment_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		log.Fatalf("准备文章插入语句失败: %v", err)
	}
	defer articleStmt.Close()

	runWorkers(ARTICLE_COUNT, workerCount, func(i int, rnd *rand.Rand) {
		topic := randomChoice(rnd, articleTopics)
		userID := rnd.Intn(USER_COUNT) + 1
		title := fmt.Sprintf("%s | %s的专栏", topic.Title, randomFullName(rnd))
		description := fmt.Sprintf("%s 来自 %s 的最新分享。", topic.Summary, randomFullName(rnd))
		content := randomArticleContent(topic, rnd)

		status := 1
		roll := rnd.Float64()
		if roll < 0.15 {
			status = 0
		} else if roll > 0.95 {
			status = 2
		}

		viewCount := rnd.Intn(8000) + 200
		likeCount := viewCount/12 + rnd.Intn(80)
		commentCount := rnd.Intn(150)

		createdAt := randomPastTime(rnd, 200)
		updatedAt := createdAt.Add(time.Duration(rnd.Intn(120)) * time.Hour)

		_, err := articleStmt.Exec(userID, title, description, content, status, viewCount, likeCount, commentCount, createdAt, updatedAt)
		if err != nil {
			log.Fatalf("插入文章数据失败: %v", err)
		}
	})

	fmt.Printf("✓ 文章数据生成完成，共 %d 条记录，耗时: %v\n", ARTICLE_COUNT, time.Since(startTime))
}

func generateResources(db *sql.DB) {
	fmt.Println("\n开始生成资源数据...")
	startTime := time.Now()

	resourceStmt, err := db.Prepare(`INSERT INTO resources (user_id, title, description, document, category_id, file_name, file_size, file_type, file_extension, file_hash, storage_path, total_chunks, download_count, view_count, like_count, comment_count, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		log.Fatalf("准备资源插入语句失败: %v", err)
	}
	defer resourceStmt.Close()

	runWorkers(RESOURCE_COUNT, workerCount, func(i int, rnd *rand.Rand) {
		seed := randomChoice(rnd, resourceSeeds)
		userID := rnd.Intn(USER_COUNT) + 1
		title := fmt.Sprintf("%s #%03d", seed.Title, rnd.Intn(900)+100)
		description := fmt.Sprintf("%s 当前版本 v%d.%d.%d。", seed.Description, rnd.Intn(3)+1, rnd.Intn(10), rnd.Intn(10))

		docBuilder := strings.Builder{}
		docBuilder.WriteString("# ")
		docBuilder.WriteString(seed.Title)
		docBuilder.WriteString("\n\n")
		docBuilder.WriteString(seed.Description)
		docBuilder.WriteString("\n\n## 更新记录\n")
		docBuilder.WriteString(fmt.Sprintf("- 维护者: %s\n", randomFullName(rnd)))
		docBuilder.WriteString(fmt.Sprintf("- 更新说明: %s\n", randomChoice(rnd, commentTemplates)))
		docBuilder.WriteString("- 版本策略: 采用 Git 标签管理，自动推送镜像\n")
		document := docBuilder.String()

		fileName := fmt.Sprintf("%s_%d%s", strings.ReplaceAll(strings.ToLower(seed.Title), " ", "_"), time.Now().UnixNano()+int64(rnd.Intn(1000)), seed.Extension)
		fileSize := int64(rnd.Intn(180)+20) * 1024 * 1024
		fileHash := randomHex(rnd, 32)
		storagePath := fmt.Sprintf("/data/resources/%d/%s", userID, fileName)
		totalChunks := 0
		if fileSize > 50*1024*1024 && rnd.Float64() < 0.4 {
			totalChunks = rnd.Intn(20) + 2
		}

		downloadCount := rnd.Intn(2500)
		viewCount := downloadCount*2 + rnd.Intn(800)
		likeCount := viewCount/18 + rnd.Intn(50)
		commentCount := rnd.Intn(90)

		status := 1
		roll := rnd.Float64()
		if roll < 0.05 {
			status = 0
		} else if roll > 0.9 {
			status = 2
		}

		createdAt := randomPastTime(rnd, 240)
		updatedAt := createdAt.Add(time.Duration(rnd.Intn(96)) * time.Hour)

		_, err := resourceStmt.Exec(userID, title, description, document, seed.CategoryID, fileName, fileSize, seed.FileType, seed.Extension, fileHash, storagePath, totalChunks, downloadCount, viewCount, likeCount, commentCount, status, createdAt, updatedAt)
		if err != nil {
			log.Fatalf("插入资源数据失败: %v", err)
		}
	})

	fmt.Printf("✓ 资源数据生成完成，共 %d 条记录，耗时: %v\n", RESOURCE_COUNT, time.Since(startTime))
}

func generateComments(db *sql.DB) {
	fmt.Println("\n开始生成评论数据...")
	startTime := time.Now()

	articleCommentStmt, err := db.Prepare(`INSERT INTO article_comments (article_id, user_id, parent_id, root_id, reply_to_user_id, content, like_count, reply_count, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		log.Fatal("准备文章评论插入语句失败: ", err)
	}
	defer articleCommentStmt.Close()

	resourceCommentStmt, err := db.Prepare(`INSERT INTO resource_comments (resource_id, user_id, parent_id, root_id, reply_to_user_id, content, like_count, reply_count, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		log.Fatal("准备资源评论插入语句失败: ", err)
	}
	defer resourceCommentStmt.Close()

	var articleCommentIDs []int64
	var resourceCommentIDs []int64
	var articleLock sync.RWMutex
	var resourceLock sync.RWMutex

	runWorkers(COMMENT_COUNT, workerCount, func(i int, rnd *rand.Rand) {
		isArticle := rnd.Float64() < 0.65
		userID := rnd.Intn(USER_COUNT) + 1
		content := fmt.Sprintf("%s —— %s", randomChoice(rnd, commentTemplates), randomFullName(rnd))
		likeCount := rnd.Intn(220)
		replyCount := rnd.Intn(50)

		status := 1
		roll := rnd.Float64()
		if roll < 0.04 {
			status = 0
		} else if roll > 0.96 {
			status = 2
		}

		createdAt := randomPastTime(rnd, 120)
		updatedAt := createdAt.Add(time.Duration(rnd.Intn(36)) * time.Hour)
		var parentID sql.NullInt64
		var rootID sql.NullInt64
		var replyTo sql.NullInt64

		if isArticle {
			articleID := rnd.Intn(ARTICLE_COUNT) + 1
			if rnd.Float64() < 0.28 {
				articleLock.RLock()
				if len(articleCommentIDs) > 0 {
					pid := articleCommentIDs[rnd.Intn(len(articleCommentIDs))]
					articleLock.RUnlock()
					parentID = sql.NullInt64{Int64: pid, Valid: true}
					rootID = parentID
					replyTo = sql.NullInt64{Int64: int64(rnd.Intn(USER_COUNT) + 1), Valid: true}
				} else {
					articleLock.RUnlock()
				}
			}

			res, err := articleCommentStmt.Exec(articleID, userID, parentID, rootID, replyTo, content, likeCount, replyCount, status, createdAt, updatedAt)
			if err != nil {
				log.Fatalf("插入文章评论失败: %v", err)
			}
			if newID, err := res.LastInsertId(); err == nil {
				articleLock.Lock()
				articleCommentIDs = append(articleCommentIDs, newID)
				articleLock.Unlock()
			}
		} else {
			resourceID := rnd.Intn(RESOURCE_COUNT) + 1
			if rnd.Float64() < 0.3 {
				resourceLock.RLock()
				if len(resourceCommentIDs) > 0 {
					pid := resourceCommentIDs[rnd.Intn(len(resourceCommentIDs))]
					resourceLock.RUnlock()
					parentID = sql.NullInt64{Int64: pid, Valid: true}
					rootID = parentID
					replyTo = sql.NullInt64{Int64: int64(rnd.Intn(USER_COUNT) + 1), Valid: true}
				} else {
					resourceLock.RUnlock()
				}
			}

			res, err := resourceCommentStmt.Exec(resourceID, userID, parentID, rootID, replyTo, content, likeCount, replyCount, status, createdAt, updatedAt)
			if err != nil {
				log.Fatalf("插入资源评论失败: %v", err)
			}
			if newID, err := res.LastInsertId(); err == nil {
				resourceLock.Lock()
				resourceCommentIDs = append(resourceCommentIDs, newID)
				resourceLock.Unlock()
			}
		}
	})

	fmt.Printf("✓ 评论数据生成完成，共 %d 条记录，耗时: %v\n", COMMENT_COUNT, time.Since(startTime))
}

func generateChatMessages(db *sql.DB) {
	fmt.Println("\n开始生成聊天消息数据...")
	startTime := time.Now()

	chatStmt, err := db.Prepare(`INSERT INTO chat_messages (user_id, username, nickname, avatar, content, message_type, send_time, ip_address, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		log.Fatalf("准备聊天消息插入语句失败: %v", err)
	}
	defer chatStmt.Close()

	runWorkers(CHAT_MESSAGE_COUNT, workerCount, func(i int, rnd *rand.Rand) {
		userID := rnd.Intn(USER_COUNT) + 1
		username := fmt.Sprintf("user_%d", userID)
		nickname := randomFullName(rnd)
		avatar := fmt.Sprintf("https://cdn.hub.local/avatar/%d.png", userID)
		content := randomChoice(rnd, messageSnippets)
		messageType := 1
		if rnd.Float64() > 0.85 {
			messageType = 2
		}

		sendTime := randomPastTime(rnd, 5)
		ipAddress := randomIP(rnd)
		status := 1
		if rnd.Float64() < 0.03 {
			status = 0
		}

		_, err := chatStmt.Exec(userID, username, nickname, avatar, content, messageType, sendTime, ipAddress, status, sendTime)
		if err != nil {
			log.Fatalf("插入聊天消息失败: %v", err)
		}
	})

	fmt.Printf("✓ 聊天消息数据生成完成，共 %d 条记录，耗时: %v\n", CHAT_MESSAGE_COUNT, time.Since(startTime))
}

func generateLikes(db *sql.DB) {
	fmt.Println("\n开始生成点赞及收藏数据...")
	startTime := time.Now()

	articleLikeStmt, err := db.Prepare(`INSERT INTO article_likes (article_id, user_id, created_at)
		VALUES (?, ?, ?)`)
	if err != nil {
		log.Fatalf("准备文章点赞语句失败: %v", err)
	}
	defer articleLikeStmt.Close()

	resourceLikeStmt, err := db.Prepare(`INSERT INTO resource_likes (resource_id, user_id, created_at)
		VALUES (?, ?, ?)`)
	if err != nil {
		log.Fatalf("准备资源点赞语句失败: %v", err)
	}
	defer resourceLikeStmt.Close()

	articleCommentLikeStmt, err := db.Prepare(`INSERT INTO article_comment_likes (comment_id, user_id, created_at)
		VALUES (?, ?, ?)`)
	if err != nil {
		log.Fatalf("准备文章评论点赞语句失败: %v", err)
	}
	defer articleCommentLikeStmt.Close()

	resourceCommentLikeStmt, err := db.Prepare(`INSERT INTO resource_comment_likes (comment_id, user_id, created_at)
		VALUES (?, ?, ?)`)
	if err != nil {
		log.Fatalf("准备资源评论点赞语句失败: %v", err)
	}
	defer resourceCommentLikeStmt.Close()

	articleIDs := fetchIDs(db, "articles")
	resourceIDs := fetchIDs(db, "resources")
	articleCommentIDs := fetchIDs(db, "article_comments")
	resourceCommentIDs := fetchIDs(db, "resource_comments")

	// 添加映射来跟踪已创建的点赞记录，防止重复
	articleLikes := make(map[string]bool)
	resourceLikes := make(map[string]bool)
	articleCommentLikes := make(map[string]bool)
	resourceCommentLikes := make(map[string]bool)

	// 添加互斥锁确保并发安全
	var articleLikesMutex sync.RWMutex
	var resourceLikesMutex sync.RWMutex
	var articleCommentLikesMutex sync.RWMutex
	var resourceCommentLikesMutex sync.RWMutex

	runWorkers(LIKE_COUNT, workerCount, func(i int, rnd *rand.Rand) {
		userID := rnd.Intn(USER_COUNT) + 1
		createdAt := randomPastTime(rnd, 150)
		roll := rnd.Float64()

		switch {
		case roll < 0.45 && len(articleIDs) > 0:
			articleID := articleIDs[rnd.Intn(len(articleIDs))]
			// 检查是否已存在相同的用户-文章点赞记录
			key := fmt.Sprintf("%d-%d", articleID, userID)
			articleLikesMutex.RLock()
			if articleLikes[key] {
				articleLikesMutex.RUnlock()
				return // 如果已存在，跳过这条记录
			}
			articleLikesMutex.RUnlock()

			// 插入前再次检查并标记
			articleLikesMutex.Lock()
			if articleLikes[key] {
				articleLikesMutex.Unlock()
				return // 双重检查，防止并发问题
			}
			articleLikes[key] = true
			articleLikesMutex.Unlock()

			if _, err := articleLikeStmt.Exec(articleID, userID, createdAt); err != nil {
				log.Fatalf("插入文章点赞失败: %v", err)
			}
		case roll < 0.7 && len(resourceIDs) > 0:
			resourceID := resourceIDs[rnd.Intn(len(resourceIDs))]
			// 检查是否已存在相同的用户-资源点赞记录
			key := fmt.Sprintf("%d-%d", resourceID, userID)
			resourceLikesMutex.RLock()
			if resourceLikes[key] {
				resourceLikesMutex.RUnlock()
				return // 如果已存在，跳过这条记录
			}
			resourceLikesMutex.RUnlock()

			// 插入前再次检查并标记
			resourceLikesMutex.Lock()
			if resourceLikes[key] {
				resourceLikesMutex.Unlock()
				return // 双重检查，防止并发问题
			}
			resourceLikes[key] = true
			resourceLikesMutex.Unlock()

			if _, err := resourceLikeStmt.Exec(resourceID, userID, createdAt); err != nil {
				log.Fatalf("插入资源点赞失败: %v", err)
			}
		case roll < 0.88 && len(articleCommentIDs) > 0:
			commentID := articleCommentIDs[rnd.Intn(len(articleCommentIDs))]
			// 检查是否已存在相同的用户-评论点赞记录
			key := fmt.Sprintf("%d-%d", commentID, userID)
			articleCommentLikesMutex.RLock()
			if articleCommentLikes[key] {
				articleCommentLikesMutex.RUnlock()
				return // 如果已存在，跳过这条记录
			}
			articleCommentLikesMutex.RUnlock()

			// 插入前再次检查并标记
			articleCommentLikesMutex.Lock()
			if articleCommentLikes[key] {
				articleCommentLikesMutex.Unlock()
				return // 双重检查，防止并发问题
			}
			articleCommentLikes[key] = true
			articleCommentLikesMutex.Unlock()

			if _, err := articleCommentLikeStmt.Exec(commentID, userID, createdAt); err != nil {
				log.Fatalf("插入文章评论点赞失败: %v", err)
			}
		case len(resourceCommentIDs) > 0:
			commentID := resourceCommentIDs[rnd.Intn(len(resourceCommentIDs))]
			// 检查是否已存在相同的用户-评论点赞记录
			key := fmt.Sprintf("%d-%d", commentID, userID)
			resourceCommentLikesMutex.RLock()
			if resourceCommentLikes[key] {
				resourceCommentLikesMutex.RUnlock()
				return // 如果已存在，跳过这条记录
			}
			resourceCommentLikesMutex.RUnlock()

			// 插入前再次检查并标记
			resourceCommentLikesMutex.Lock()
			if resourceCommentLikes[key] {
				resourceCommentLikesMutex.Unlock()
				return // 双重检查，防止并发问题
			}
			resourceCommentLikes[key] = true
			resourceCommentLikesMutex.Unlock()

			if _, err := resourceCommentLikeStmt.Exec(commentID, userID, createdAt); err != nil {
				log.Fatalf("插入资源评论点赞失败: %v", err)
			}
		default:
			// 若暂时没有目标ID则跳过
		}
	})

	fmt.Printf("✓ 点赞数据生成完成，共 %d 条记录，耗时: %v\n", LIKE_COUNT, time.Since(startTime))
}

func generateLoginHistory(db *sql.DB) {
	fmt.Println("\n开始生成登录历史数据...")
	startTime := time.Now()

	loginStmt, err := db.Prepare(`INSERT INTO user_login_history (user_id, username, login_time, login_ip, user_agent, login_status, province, city, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		log.Fatalf("准备登录历史插入语句失败: %v", err)
	}
	defer loginStmt.Close()

	runWorkers(LOGIN_HISTORY_COUNT, workerCount, func(i int, rnd *rand.Rand) {
		userID := rnd.Intn(USER_COUNT) + 1
		username := fmt.Sprintf("user_%d", userID)
		loginTime := randomPastTime(rnd, 120)
		loginIP := randomIP(rnd)
		userAgent := randomChoice(rnd, loginUserAgents)

		loginStatus := 1
		if rnd.Float64() < 0.08 {
			loginStatus = 0
		}

		province := randomChoice(rnd, provinceList)
		city := randomChoice(rnd, provinceCities[province])

		_, err := loginStmt.Exec(userID, username, loginTime, loginIP, userAgent, loginStatus, province, city, loginTime)
		if err != nil {
			log.Fatalf("插入登录历史失败: %v", err)
		}
	})

	fmt.Printf("✓ 登录历史数据生成完成，共 %d 条记录，耗时: %v\n", LOGIN_HISTORY_COUNT, time.Since(startTime))
}

func generateStatistics(db *sql.DB) {
	fmt.Println("\n开始生成统计数据...")
	startTime := time.Now()

	userStatStmt, err := db.Prepare(`INSERT INTO user_statistics (date, login_count, register_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		log.Fatalf("准备用户统计语句失败: %v", err)
	}
	defer userStatStmt.Close()

	apiStatStmt, err := db.Prepare(`INSERT INTO api_statistics (date, endpoint, method, success_count, error_count, total_count, avg_latency_ms, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		log.Fatal("准备 API 统计语句失败:", err)
	}
	defer apiStatStmt.Close()

	endpoints := []string{
		"/api/users/login",
		"/api/users/register",
		"/api/articles",
		"/api/articles/{id}",
		"/api/resources",
		"/api/resources/{id}",
		"/api/chat/messages",
		"/api/comments",
	}
	methods := []string{"GET", "POST", "PUT", "DELETE"}

	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < STATISTICS_COUNT; i++ {
		day := time.Now().AddDate(0, 0, -i)
		date := day.Format("2006-01-02")
		loginCount := 250 + rnd.Intn(250)
		registerCount := 15 + rnd.Intn(40)
		createdAt := day

		if _, err := userStatStmt.Exec(date, loginCount, registerCount, createdAt, createdAt); err != nil {
			log.Fatalf("插入用户统计失败: %v", err)
		}

		for j, endpoint := range endpoints {
			method := methods[(i+j)%len(methods)]
			successCount := 400 + rnd.Intn(900)
			errorCount := rnd.Intn(30)
			totalCount := successCount + errorCount
			avgLatency := 50 + rnd.Float64()*420

			if _, err := apiStatStmt.Exec(date, endpoint, method, successCount, errorCount, totalCount, avgLatency, createdAt, createdAt); err != nil {
				log.Fatalf("插入 API 统计失败: %v", err)
			}
		}
	}

	fmt.Printf("✓ 统计数据生成完成，共 %d 天的数据，耗时: %v\n", STATISTICS_COUNT, time.Since(startTime))
}
