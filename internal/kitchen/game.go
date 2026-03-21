package kitchen

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	boardWidth        = 34
	boardHeight       = 16
	tickRate          = 220 * time.Millisecond
	maxPlayers        = 4
	maxCPU            = 100
	maxMemory         = 12
	maxProcessSlots   = 4
	orderTimeLimit    = 75 * time.Second
	anomalyTickMin    = 18
	anomalyTickSpread = 8
)

type point struct {
	X int
	Y int
}

type itemType string

const (
	itemNone        itemType = ""
	itemSecurityLib itemType = "security_lib"
	itemCoreUpdate  itemType = "core_update"
	itemParser      itemType = "parser"
	itemStdout      itemType = "stdout_driver"
	itemCommandMap  itemType = "command_map"
	itemNullGuard   itemType = "null_guard"
)

type stationKind string

const (
	stationSecurity stationKind = "security"
	stationCore     stationKind = "core"
	stationParser   stationKind = "parser"
	stationStdout   stationKind = "stdout"
	stationMixer    stationKind = "mixer"
	stationBuild    stationKind = "build"
	stationDebug    stationKind = "debug"
	stationDispatch stationKind = "dispatch"
	stationVirus    stationKind = "virus"
	stationFreeze   stationKind = "freeze"
)

type playerState struct {
	player     *Player
	pos        point
	carrying   itemType
	score      int
	deliveries int
	mistakes   int
	prompt     *promptState
}

type promptState struct {
	kind    string
	station stationKind
	input   string
	target  string
	hint    string
}

type orderTemplate struct {
	ID       string
	Title    string
	Result   string
	Requires map[itemType]int
	Code     []string
	BugLine  int
	Fix      string
	FixLabel string
}

type orderState struct {
	orderTemplate
	queuedAt      time.Time
	codeFixed     bool
	buildProgress map[itemType]int
	wrongDrops    int
}

type anomalyState struct {
	kind    stationKind
	active  bool
	target  stationKind
	command string
	hint    string
}

type inputEvent struct {
	nick string
	key  string
}

type joinEvent struct {
	nick   string
	player *Player
	reply  chan error
}

type Game struct {
	mu sync.Mutex

	players   map[string]*playerState
	joinOrder []string
	inputCh   chan inputEvent
	joinCh    chan joinEvent
	leaveCh   chan string
	rng       *rand.Rand

	activeOrder *orderState
	nextOrders  []*orderState
	streak      int
	totalScore  int
	cpu         int
	memory      int
	processes   int
	ticks       int
	gameOver    bool
	message     string
	mixBuffer   []itemType
	mixOutput   itemType
	anomalies   map[stationKind]*anomalyState
}

var stationPositions = map[stationKind]point{
	stationSecurity: {3, 3},
	stationCore:     {3, 6},
	stationParser:   {3, 9},
	stationStdout:   {3, 12},
	stationMixer:    {14, 4},
	stationBuild:    {14, 9},
	stationDebug:    {14, 13},
	stationDispatch: {27, 9},
	stationVirus:    {27, 4},
	stationFreeze:   {27, 13},
}

var orderTemplates = []orderTemplate{
	{
		ID:     "security_patch",
		Title:  "security patch",
		Result: "deploy a hardened patch",
		Requires: map[itemType]int{
			itemSecurityLib: 3,
			itemCoreUpdate:  1,
		},
		Code: []string{
			"LOAD security_lib",
			"APPLY core_update",
			"ALLOW_ALL = true",
			"COMMIT /prod",
		},
		BugLine:  2,
		Fix:      "comment allow_all",
		FixLabel: "закомментировать небезопасную строку",
	},
	{
		ID:     "cli_tool",
		Title:  "cli tool",
		Result: "ship a tiny command-line utility",
		Requires: map[itemType]int{
			itemParser:     1,
			itemStdout:     1,
			itemCommandMap: 1,
		},
		Code: []string{
			"BOOT parser",
			"ROUTE -> nil",
			"BIND stdout_driver",
			"BUILD cli_tool",
		},
		BugLine:  1,
		Fix:      "replace nil route",
		FixLabel: "починить пустой маршрут",
	},
	{
		ID:     "hotfix",
		Title:  "hotfix",
		Result: "push an urgent midnight fix",
		Requires: map[itemType]int{
			itemParser:     1,
			itemCoreUpdate: 1,
			itemNullGuard:  1,
		},
		Code: []string{
			"SCAN logs",
			"IF target != nil",
			"  RETURN panic",
			"EMIT hotfix",
		},
		BugLine:  2,
		Fix:      "add nil-check",
		FixLabel: "убрать панику и добавить guard",
	},
	{
		ID:     "monitor_daemon",
		Title:  "monitor daemon",
		Result: "assemble a watcher for noisy systems",
		Requires: map[itemType]int{
			itemSecurityLib: 1,
			itemStdout:      1,
			itemCommandMap:  1,
		},
		Code: []string{
			"AUTH security_lib",
			"PRINT debug_dump",
			"MAP commands",
			"RUN monitor_daemon",
		},
		BugLine:  1,
		Fix:      "remove debug print",
		FixLabel: "убрать лишний debug output",
	},
}

func NewGame() *Game {
	g := &Game{
		players:   make(map[string]*playerState),
		inputCh:   make(chan inputEvent, 64),
		joinCh:    make(chan joinEvent, 8),
		leaveCh:   make(chan string, 8),
		rng:       rand.New(rand.NewSource(time.Now().UnixNano())),
		anomalies: make(map[stationKind]*anomalyState),
		message:   "добро пожаловать на цифровую кухню",
	}
	g.activeOrder = g.newOrder()
	g.nextOrders = []*orderState{g.newOrder(), g.newOrder()}
	g.scheduleAnomalies()
	go g.loop()
	return g
}

func (g *Game) Join(nick string, p *Player) error {
	reply := make(chan error, 1)
	g.joinCh <- joinEvent{nick: nick, player: p, reply: reply}
	return <-reply
}

func (g *Game) Leave(nick string) { g.leaveCh <- nick }

func (g *Game) Input(nick, key string) {
	select {
	case g.inputCh <- inputEvent{nick: nick, key: key}:
	default:
	}
}

func (g *Game) loop() {
	ticker := time.NewTicker(tickRate)
	defer ticker.Stop()
	for {
		select {
		case ev := <-g.joinCh:
			g.mu.Lock()
			err := g.handleJoin(ev.nick, ev.player)
			g.broadcastLocked()
			g.mu.Unlock()
			ev.reply <- err
		case nick := <-g.leaveCh:
			g.mu.Lock()
			g.handleLeave(nick)
			g.broadcastLocked()
			g.mu.Unlock()
		case ev := <-g.inputCh:
			g.mu.Lock()
			g.handleInput(ev.nick, ev.key)
			g.broadcastLocked()
			g.mu.Unlock()
		case <-ticker.C:
			g.mu.Lock()
			g.tick()
			g.broadcastLocked()
			g.mu.Unlock()
		}
	}
}

func (g *Game) handleJoin(nick string, p *Player) error {
	if _, exists := g.players[nick]; exists {
		return fmt.Errorf("nick %q is already online", nick)
	}
	if len(g.players) >= maxPlayers {
		return fmt.Errorf("lobby is full")
	}
	spawn := []point{{7, 4}, {7, 7}, {7, 10}, {7, 13}}[len(g.players)]
	g.players[nick] = &playerState{player: p, pos: spawn}
	g.joinOrder = append(g.joinOrder, nick)
	g.message = fmt.Sprintf("%s подключился к кухне", nick)
	return nil
}

func (g *Game) handleLeave(nick string) {
	if p, ok := g.players[nick]; ok {
		p.player.close()
		delete(g.players, nick)
	}
	filtered := g.joinOrder[:0]
	for _, existing := range g.joinOrder {
		if existing != nick {
			filtered = append(filtered, existing)
		}
	}
	g.joinOrder = filtered
	g.message = fmt.Sprintf("%s вышел из смены", nick)
}

func (g *Game) handleInput(nick, key string) {
	p := g.players[nick]
	if p == nil {
		return
	}
	if g.gameOver {
		if key == "r" || key == "R" {
			g.reset()
		}
		return
	}
	if p.prompt != nil {
		g.handlePromptInput(p, key)
		return
	}

	switch key {
	case "up", "w", "W", "ц", "Ц":
		g.movePlayer(p, 0, -1)
	case "down", "s", "S", "ы", "Ы":
		g.movePlayer(p, 0, 1)
	case "left", "a", "A", "ф", "Ф":
		g.movePlayer(p, -1, 0)
	case "right", "d", "D", "в", "В":
		g.movePlayer(p, 1, 0)
	case "e", "E", "у", "У", "enter":
		g.interact(nick, p)
	case "x", "X", "ч", "Ч":
		if p.carrying != itemNone {
			p.carrying = itemNone
			g.bumpCPU(1)
			g.message = fmt.Sprintf("%s очистил слот и выбросил ингредиент", nick)
		}
	}
}

func (g *Game) handlePromptInput(p *playerState, key string) {
	switch key {
	case "esc":
		p.prompt = nil
		g.message = fmt.Sprintf("%s отменил команду", p.player.Nick)
		return
	case "enter":
		input := strings.TrimSpace(strings.ToLower(p.prompt.input))
		if input == p.prompt.target {
			g.resolvePrompt(p)
		} else {
			p.mistakes++
			g.bumpCPU(5)
			g.message = fmt.Sprintf("%s ввёл неверную команду", p.player.Nick)
		}
		p.prompt = nil
		return
	case "backspace":
		if len(p.prompt.input) > 0 {
			p.prompt.input = p.prompt.input[:len(p.prompt.input)-1]
		}
		return
	}
	if len(key) == 1 && key[0] >= 32 && key[0] < 127 {
		if len(p.prompt.input) < 32 {
			p.prompt.input += strings.ToLower(key)
		}
	}
}

func (g *Game) resolvePrompt(p *playerState) {
	switch p.prompt.kind {
	case "debug":
		g.activeOrder.codeFixed = true
		g.totalScore += 40
		g.message = fmt.Sprintf("%s исправил код-рецепт", p.player.Nick)
	case "virus":
		if a := g.anomalies[stationVirus]; a != nil {
			a.active = false
			g.totalScore += 20
			g.message = fmt.Sprintf("%s почистил вирус на %s", p.player.Nick, stationLabel(a.target))
		}
	case "freeze":
		if a := g.anomalies[stationFreeze]; a != nil {
			a.active = false
			g.totalScore += 20
			g.message = fmt.Sprintf("%s разморозил %s", p.player.Nick, stationLabel(a.target))
		}
	}
}

func (g *Game) movePlayer(p *playerState, dx, dy int) {
	next := point{p.pos.X + dx, p.pos.Y + dy}
	if next.X < 1 || next.X > boardWidth || next.Y < 1 || next.Y > boardHeight {
		return
	}
	p.pos = next
}

func (g *Game) interact(nick string, p *playerState) {
	station, ok := stationAt(p.pos)
	if !ok {
		g.message = "здесь нет рабочей станции"
		return
	}
	if g.stationFrozen(station) || g.stationVirusBlocked(station) {
		g.bumpCPU(2)
		g.message = fmt.Sprintf("%s недоступна из-за аномалии", stationLabel(station))
		return
	}

	switch station {
	case stationSecurity, stationCore, stationParser, stationStdout:
		g.pickBaseIngredient(nick, p, station)
	case stationMixer:
		g.interactMixer(nick, p)
	case stationBuild:
		g.interactBuild(nick, p)
	case stationDebug:
		g.interactDebug(nick, p)
	case stationDispatch:
		g.interactDispatch(nick, p)
	case stationVirus:
		g.interactAnomalyPrompt(p, stationVirus)
	case stationFreeze:
		g.interactAnomalyPrompt(p, stationFreeze)
	}
}

func (g *Game) pickBaseIngredient(nick string, p *playerState, station stationKind) {
	if p.carrying != itemNone {
		g.message = fmt.Sprintf("%s уже несёт %s", nick, itemLabel(p.carrying))
		return
	}
	item := sourceItem(station)
	if g.memory >= maxMemory {
		g.bumpCPU(3)
		g.message = "память переполнена — сначала разгрузите кухню"
		return
	}
	p.carrying = item
	g.message = fmt.Sprintf("%s взял %s", nick, itemLabel(item))
}

func (g *Game) interactMixer(nick string, p *playerState) {
	if p.carrying != itemNone {
		if len(g.mixBuffer) >= 2 || g.mixOutput != itemNone {
			g.bumpCPU(1)
			g.message = "сборочный слот занят"
			return
		}
		g.mixBuffer = append(g.mixBuffer, p.carrying)
		g.message = fmt.Sprintf("%s положил %s в компоновщик", nick, itemLabel(p.carrying))
		p.carrying = itemNone
		if out, ok := mixResult(g.mixBuffer); ok {
			g.mixBuffer = nil
			g.mixOutput = out
			g.totalScore += 10
			g.message = fmt.Sprintf("компоновщик собрал %s", itemLabel(out))
		} else if len(g.mixBuffer) == 2 {
			g.mixBuffer = nil
			g.bumpCPU(4)
			g.message = "ошибка сборки: несовместимые модули"
		}
		return
	}
	if g.mixOutput != itemNone {
		p.carrying = g.mixOutput
		g.mixOutput = itemNone
		g.message = fmt.Sprintf("%s забрал %s", nick, itemLabel(p.carrying))
		return
	}
	g.message = "компоновщик ждёт два совместимых ингредиента"
}

func (g *Game) interactBuild(nick string, p *playerState) {
	if g.activeOrder == nil {
		return
	}
	if p.carrying == itemNone {
		g.activeOrder.buildProgress = make(map[itemType]int)
		g.bumpCPU(2)
		g.message = fmt.Sprintf("%s очистил рабочую сборку", nick)
		return
	}
	need := g.activeOrder.Requires[p.carrying]
	current := g.activeOrder.buildProgress[p.carrying]
	item := p.carrying
	if need == 0 || current >= need {
		g.activeOrder.wrongDrops++
		p.mistakes++
		p.carrying = itemNone
		g.bumpCPU(4)
		g.message = fmt.Sprintf("%s добавил лишний %s", nick, itemLabel(item))
		return
	}
	g.activeOrder.buildProgress[item]++
	g.message = fmt.Sprintf("%s добавил %s к заказу", nick, itemLabel(item))
	p.carrying = itemNone
}

func (g *Game) interactDebug(nick string, p *playerState) {
	if g.activeOrder == nil {
		return
	}
	if g.activeOrder.codeFixed {
		g.message = "код-рецепт уже исправлен"
		return
	}
	p.prompt = &promptState{
		kind:    "debug",
		station: stationDebug,
		input:   "",
		target:  g.activeOrder.Fix,
		hint:    g.activeOrder.FixLabel,
	}
	g.message = fmt.Sprintf("%s открыл терминал отладки", nick)
}

func (g *Game) interactDispatch(nick string, p *playerState) {
	if g.activeOrder == nil {
		return
	}
	if !g.orderReady(g.activeOrder) {
		g.bumpCPU(6)
		g.streak = 0
		p.mistakes++
		g.message = "заказ не готов: не хватает ингредиентов или код не исправлен"
		return
	}
	elapsed := time.Since(g.activeOrder.queuedAt)
	speedBonus := max(0, 80-int(elapsed.Seconds()))
	accuracyBonus := max(0, 30-g.activeOrder.wrongDrops*10)
	streakBonus := g.streak * 15
	points := 120 + speedBonus + accuracyBonus + streakBonus
	g.totalScore += points
	g.streak++
	p.score += points
	p.deliveries++
	g.cpu = max(0, g.cpu-12)
	g.message = fmt.Sprintf("%s закрыл заказ %s (+%d)", nick, g.activeOrder.Title, points)
	g.activeOrder = g.nextOrders[0]
	g.nextOrders = []*orderState{g.nextOrders[1], g.newOrder()}
}

func (g *Game) interactAnomalyPrompt(p *playerState, kind stationKind) {
	a := g.anomalies[kind]
	if a == nil || !a.active {
		g.message = fmt.Sprintf("станция %s в норме", stationLabel(kind))
		return
	}
	p.prompt = &promptState{kind: string(kind), station: kind, target: a.command, hint: a.hint}
	g.message = fmt.Sprintf("%s: введите %q", stationLabel(kind), a.command)
}

func (g *Game) tick() {
	if g.gameOver {
		return
	}
	g.ticks++
	g.scheduleAnomalies()
	g.cpu += 1
	if g.activeOrder != nil {
		if overdue := time.Since(g.activeOrder.queuedAt) - orderTimeLimit; overdue > 0 {
			g.cpu += 2
		}
		if !g.activeOrder.codeFixed {
			g.cpu += 1
		}
	}
	for _, a := range g.anomalies {
		if a.active {
			g.cpu += 1
		}
	}
	g.memory = g.computeMemoryUsage()
	g.processes = g.computeProcessUsage()
	if g.memory > maxMemory {
		g.cpu += 4
	}
	if g.processes > maxProcessSlots {
		g.cpu += 3
	}
	if g.cpu >= maxCPU {
		g.cpu = maxCPU
		g.gameOver = true
		g.message = "kernel panic: кухня перегружена"
	}
}

func (g *Game) computeMemoryUsage() int {
	used := len(g.mixBuffer)
	if g.mixOutput != itemNone {
		used++
	}
	if g.activeOrder != nil {
		for _, c := range g.activeOrder.buildProgress {
			used += c
		}
	}
	for _, p := range g.players {
		if p.carrying != itemNone {
			used++
		}
	}
	return used
}

func (g *Game) computeProcessUsage() int {
	used := 1
	if g.activeOrder != nil && !g.activeOrder.codeFixed {
		used++
	}
	for _, a := range g.anomalies {
		if a.active {
			used++
		}
	}
	if len(g.mixBuffer) > 0 || g.mixOutput != itemNone {
		used++
	}
	return used
}

func (g *Game) scheduleAnomalies() {
	if g.ticks == 0 || g.ticks%(anomalyTickMin+g.rng.Intn(anomalyTickSpread)) != 0 {
		return
	}
	kind := stationVirus
	if g.rng.Intn(2) == 0 {
		kind = stationFreeze
	}
	a := g.anomalies[kind]
	if a != nil && a.active {
		return
	}
	targets := []stationKind{stationSecurity, stationCore, stationParser, stationStdout, stationMixer, stationBuild, stationDebug}
	target := targets[g.rng.Intn(len(targets))]
	if kind == stationVirus {
		g.anomalies[kind] = &anomalyState{kind: kind, active: true, target: target, command: "scan --clean", hint: "антивирусная чистка"}
		g.message = fmt.Sprintf("вирус заражает %s", stationLabel(target))
	} else {
		g.anomalies[kind] = &anomalyState{kind: kind, active: true, target: target, command: "thaw --force", hint: "разморозить станцию"}
		g.message = fmt.Sprintf("freeze заморозил %s", stationLabel(target))
	}
}

func (g *Game) stationFrozen(station stationKind) bool {
	a := g.anomalies[stationFreeze]
	return a != nil && a.active && a.target == station
}

func (g *Game) stationVirusBlocked(station stationKind) bool {
	a := g.anomalies[stationVirus]
	return a != nil && a.active && a.target == station
}

func (g *Game) bumpCPU(delta int) {
	g.cpu += delta
	if g.cpu > maxCPU {
		g.cpu = maxCPU
	}
}

func (g *Game) orderReady(order *orderState) bool {
	if order == nil || !order.codeFixed {
		return false
	}
	for item, need := range order.Requires {
		if order.buildProgress[item] != need {
			return false
		}
	}
	return true
}

func (g *Game) newOrder() *orderState {
	t := orderTemplates[g.rng.Intn(len(orderTemplates))]
	copyReq := make(map[itemType]int, len(t.Requires))
	for k, v := range t.Requires {
		copyReq[k] = v
	}
	code := make([]string, len(t.Code))
	copy(code, t.Code)
	return &orderState{orderTemplate: orderTemplate{ID: t.ID, Title: t.Title, Result: t.Result, Requires: copyReq, Code: code, BugLine: t.BugLine, Fix: t.Fix, FixLabel: t.FixLabel}, queuedAt: time.Now(), buildProgress: make(map[itemType]int)}
}

func (g *Game) reset() {
	g.activeOrder = g.newOrder()
	g.nextOrders = []*orderState{g.newOrder(), g.newOrder()}
	g.mixBuffer = nil
	g.mixOutput = itemNone
	g.streak = 0
	g.totalScore = 0
	g.cpu = 0
	g.memory = 0
	g.processes = 1
	g.gameOver = false
	g.ticks = 0
	g.anomalies = make(map[stationKind]*anomalyState)
	g.message = "новая смена началась"
	for _, p := range g.players {
		p.carrying = itemNone
		p.prompt = nil
		p.score = 0
		p.deliveries = 0
		p.mistakes = 0
	}
}

func (g *Game) snapshot() frameState {
	players := make([]playerFrame, 0, len(g.players))
	for _, nick := range g.joinOrder {
		p := g.players[nick]
		if p == nil {
			continue
		}
		players = append(players, playerFrame{Nick: nick, X: p.pos.X, Y: p.pos.Y, Carrying: p.carrying, Score: p.score, Deliveries: p.deliveries, Mistakes: p.mistakes, Prompt: p.prompt})
	}
	requires := make([]requirementFrame, 0, len(g.activeOrder.Requires))
	for item, need := range g.activeOrder.Requires {
		requires = append(requires, requirementFrame{Item: item, Need: need, Have: g.activeOrder.buildProgress[item]})
	}
	sort.Slice(requires, func(i, j int) bool { return requires[i].Item < requires[j].Item })
	next := make([]string, 0, len(g.nextOrders))
	for _, o := range g.nextOrders {
		next = append(next, o.Title)
	}
	anomalies := make([]anomalyFrame, 0, len(g.anomalies))
	for _, k := range []stationKind{stationVirus, stationFreeze} {
		if a := g.anomalies[k]; a != nil && a.active {
			anomalies = append(anomalies, anomalyFrame{Kind: k, Target: a.target, Command: a.command})
		}
	}
	code := make([]string, len(g.activeOrder.Code))
	copy(code, g.activeOrder.Code)
	return frameState{Players: players, CPU: g.cpu, Memory: g.memory, Processes: g.processes, TotalScore: g.totalScore, Streak: g.streak, Message: g.message, MixBuffer: append([]itemType(nil), g.mixBuffer...), MixOutput: g.mixOutput, ActiveOrder: orderFrame{Title: g.activeOrder.Title, Result: g.activeOrder.Result, Requires: requires, Code: code, BugLine: g.activeOrder.BugLine, CodeFixed: g.activeOrder.codeFixed, WrongDrops: g.activeOrder.wrongDrops, Age: time.Since(g.activeOrder.queuedAt)}, NextOrders: next, Anomalies: anomalies, GameOver: g.gameOver}
}

func (g *Game) broadcastLocked() {
	state := g.snapshot()
	for _, nick := range g.joinOrder {
		p := g.players[nick]
		if p == nil {
			continue
		}
		p.player.enqueue(renderFrame(state, nick))
	}
}

func stationAt(pos point) (stationKind, bool) {
	for station, stationPos := range stationPositions {
		if stationPos == pos {
			return station, true
		}
	}
	return "", false
}

func sourceItem(station stationKind) itemType {
	switch station {
	case stationSecurity:
		return itemSecurityLib
	case stationCore:
		return itemCoreUpdate
	case stationParser:
		return itemParser
	default:
		return itemStdout
	}
}

func mixResult(items []itemType) (itemType, bool) {
	if len(items) != 2 {
		return itemNone, false
	}
	copyItems := append([]itemType(nil), items...)
	sort.Slice(copyItems, func(i, j int) bool { return copyItems[i] < copyItems[j] })
	switch {
	case copyItems[0] == itemParser && copyItems[1] == itemStdout:
		return itemCommandMap, true
	case copyItems[0] == itemCoreUpdate && copyItems[1] == itemParser:
		return itemNullGuard, true
	default:
		return itemNone, false
	}
}

func itemLabel(item itemType) string {
	switch item {
	case itemSecurityLib:
		return "security_lib"
	case itemCoreUpdate:
		return "core_update"
	case itemParser:
		return "parser"
	case itemStdout:
		return "stdout_driver"
	case itemCommandMap:
		return "command_map"
	case itemNullGuard:
		return "null_guard"
	default:
		return "пусто"
	}
}

func stationLabel(station stationKind) string {
	switch station {
	case stationSecurity:
		return "security locker"
	case stationCore:
		return "core cache"
	case stationParser:
		return "parser rack"
	case stationStdout:
		return "stdout shelf"
	case stationMixer:
		return "compiler oven"
	case stationBuild:
		return "build bench"
	case stationDebug:
		return "debug terminal"
	case stationDispatch:
		return "dispatch gate"
	case stationVirus:
		return "anti-virus"
	case stationFreeze:
		return "de-freezer"
	default:
		return string(station)
	}
}
