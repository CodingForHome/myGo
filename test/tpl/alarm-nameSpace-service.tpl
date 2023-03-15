{
  "elements": [
    {
      "tag": "column_set",
      "flex_mode": "none",
      "background_style": "grey",
      "columns": [
        {
          "tag": "column",
          "width": "weighted",
          "weight": 2,
          "vertical_align": "top",
          "elements": [
            {
              "tag": "div",
              "text": {
                "content": "**🕐 卡片发送时间:** {{.CardSendTime}}",
                "tag": "lark_md"
              }
            }
          ]
        },
        {
          "tag": "column",
          "width": "weighted",
          "weight": 1,
          "vertical_align": "top",
          "elements": [
            {
              "tag": "div",
              "text": {
                "content": "**🌋 NameSpace:** {{.NameSpace}}",
                "tag": "lark_md"
              }
            }
          ]
        }
      ]
    },
    {
      "tag": "hr"
    },
    {
      "tag": "markdown",
      "content": "# ⏰ Total:\n{{.Total}}\n# 📋 Base:level & nameSpace:\n[点我查看]({{.LevelNameSpace}})\n# {{range .Services}} 💁‍♂️ {{.ServiceName}}(Total:{{.Total}}, Display up to 5）:\n {{range $index, $value := .Link}}\n<{{$index}}> : [点我查看] ({{$value}})\n{{end}}{{end}}"
    }
  ],
  "header": {
    "template": "red",
    "title": {
      "content": "🔥 业务告警",
      "tag": "plain_text"
    }
  }
}