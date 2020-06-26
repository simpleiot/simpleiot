module Pages.Top exposing (Flags, Model, Msg, page)

import Data.Device as D
import Data.Sample exposing (Sample, renderSample)
import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Element.Input as Input
import Global
import Page exposing (Document, Page)
import Time
import UI.Icon as Icon
import UI.Style as Style exposing (colors, size)


type alias Flags =
    ()


type alias DeviceEdit =
    { id : String
    , config : D.Config
    }


type alias Model =
    { deviceEdit : Maybe DeviceEdit
    }


type Msg
    = EditDeviceDescription String String
    | PostConfig String D.Config
    | DiscardEditedDeviceDescription
    | DeleteDevice String
    | Tick Time.Posix


page : Page Flags Model Msg
page =
    Page.component
        { init = init
        , update = update
        , subscriptions = subscriptions
        , view = view
        }


init : Global.Model -> Flags -> ( Model, Cmd Msg, Cmd Global.Msg )
init _ _ =
    ( Model Nothing, Cmd.none, Global.send Global.RequestDevices )


update : Global.Model -> Msg -> Model -> ( Model, Cmd Msg, Cmd Global.Msg )
update global msg model =
    case msg of
        EditDeviceDescription id description ->
            ( { model | deviceEdit = Just { id = id, config = { description = description } } }
            , Cmd.none
            , Cmd.none
            )

        PostConfig id config ->
            ( { model | deviceEdit = Nothing }
            , Cmd.none
            , Global.send <| Global.UpdateDeviceConfig id config
            )

        DiscardEditedDeviceDescription ->
            ( { model | deviceEdit = Nothing }
            , Cmd.none
            , Cmd.none
            )

        DeleteDevice id ->
            ( model, Cmd.none, Global.send <| Global.DeleteDevice id )

        Tick _ ->
            ( model
            , Cmd.none
            , case global.auth of
                Global.SignedIn _ ->
                    Global.send Global.RequestDevices

                Global.SignedOut _ ->
                    Cmd.none
            )


subscriptions : Global.Model -> Model -> Sub Msg
subscriptions _ _ =
    Sub.batch
        [ Time.every 5000 Tick
        ]


view : Global.Model -> Model -> Document Msg
view global model =
    { title = "SIOT Devices"
    , body =
        [ column
            [ width fill, spacing 32 ]
            [ el Style.h2 <| text "Devices"
            , case global.auth of
                Global.SignedIn sess ->
                    viewDevices sess.data.devices model sess.isRoot

                _ ->
                    el [ padding 16 ] <| text "Sign in to view your devices."
            ]
        ]
    }


viewDevices : List D.Device -> Model -> Bool -> Element Msg
viewDevices devices model isRoot =
    column
        [ width fill
        , spacing 24
        ]
    <|
        List.map
            (\d ->
                viewDevice d.mod d.device isRoot
            )
        <|
            mergeDeviceEdit devices model.deviceEdit


type alias DeviceMod =
    { device : D.Device
    , mod : Bool
    }


mergeDeviceEdit : List D.Device -> Maybe DeviceEdit -> List DeviceMod
mergeDeviceEdit devices devConfigEdit =
    case devConfigEdit of
        Just edit ->
            List.map
                (\d ->
                    if edit.id == d.id then
                        { device = { d | config = edit.config }, mod = True }

                    else
                        { device = d, mod = False }
                )
                devices

        Nothing ->
            List.map (\d -> { device = d, mod = False }) devices


viewDevice : Bool -> D.Device -> Bool -> Element Msg
viewDevice mod device isRoot =
    let
        sysStateIcon =
            case device.state.sysState of
                -- not sure who D.sysStatePowerOff does not work here ...
                1 ->
                    Icon.power

                2 ->
                    Icon.cloudOff

                3 ->
                    Icon.cloud

                _ ->
                    Element.none

        background =
            case device.state.sysState of
                3 ->
                    Style.colors.white

                _ ->
                    Style.colors.gray
    in
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color colors.black
        , Background.color background
        , spacing 6
        ]
        [ wrappedRow [ spacing 10 ]
            [ sysStateIcon
            , viewDeviceId device.id
            , if isRoot then
                Icon.x (DeleteDevice device.id)

              else
                Element.none
            , Input.text
                [ Background.color background ]
                { onChange = \d -> EditDeviceDescription device.id d
                , text = device.config.description
                , placeholder = Just <| Input.placeholder [] <| text "device description"
                , label = Input.labelHidden "device description"
                }
            , if mod then
                Icon.check (PostConfig device.id device.config)

              else
                Element.none
            , if mod then
                Icon.x DiscardEditedDeviceDescription

              else
                Element.none
            ]
        , viewIoList device.state.ios
        ]


viewDeviceId : String -> Element Msg
viewDeviceId id =
    el
        [ padding 16
        , size.heading
        ]
    <|
        text id


viewIoList : List Sample -> Element Msg
viewIoList ios =
    column
        [ padding 16
        , spacing 6
        ]
    <|
        List.map (renderSample >> text) ios
